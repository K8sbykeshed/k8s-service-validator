package suites

import (
	"context"
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
)

func mustOrFatal(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

// Pods switch to use specified set of clusterIPs,
// either disabledClusterIPs or toggledClusterIPs for reachability access
func podsUseClusterIPs(pods []*entities.Pod, ips []string) error {
	if len(pods) != len(ips) {
		return errors.New("pods number does not match the provided IP number")
	}
	for i, pod := range pods {
		pod.SetClusterIP(ips[i])
	}
	return nil
}

// Services switch to use specified label, with key and value
func servicesSetLabel(services kubernetes.Services, key, value string) error {
	for _, service := range services {
		if err := service.SetLabel(key, value); err != nil {
			return err
		}
	}
	return nil
}

// Services switch to not having the label with key
func servicesRemoveLabel(services kubernetes.Services, key string) error {
	for _, service := range services {
		if err := service.RemoveLabel(key); err != nil {
			return err
		}
	}
	return nil
}

func TestProxyName(t *testing.T) {
	pods := model.AllPods()
	var disabledServices kubernetes.Services
	var toggledServices kubernetes.Services

	var disabledClusterIPs []string
	var toggledClusterIPs []string

	labelKey := "service.kubernetes.io/service-proxy-name"
	labelValue := "foo-bar"

	featureProxyName := features.New("ProxyName").WithLabel("type", "proxyName").
		Setup(func(_ context.Context, t *testing.T, _ *envconf.Config) context.Context {
			for _, pod := range pods {
				var (
					disabledClusterIP string
					toggledClusterIP  string
					err               error
				)
				// create a disabled service labeled with service-proxy-name
				var serviceDisabled kubernetes.ServiceBase = kubernetes.NewService(cs, pod.ClusterIPService())
				if _, err := serviceDisabled.Create(); err != nil {
					t.Fatal()
				}
				disabledServices = append(disabledServices, serviceDisabled.(*kubernetes.Service))

				// create a service without the label, but will be toggled later
				var serviceToggled kubernetes.ServiceBase = kubernetes.NewService(cs, pod.ClusterIPService())
				if _, err := serviceToggled.Create(); err != nil {
					t.Fatal()
				}
				toggledServices = append(toggledServices, serviceToggled.(*kubernetes.Service))

				// wait for the disabled service
				if result, err := serviceDisabled.WaitForEndpoint(); err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}
				if disabledClusterIP, err = serviceDisabled.WaitForClusterIP(); err != nil || disabledClusterIP == "" {
					t.Fatal(errors.New("no cluster IP available"))
				}

				// wait for the toggled service
				if result, err := serviceToggled.WaitForEndpoint(); err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}
				if toggledClusterIP, err = serviceToggled.WaitForClusterIP(); err != nil || toggledClusterIP == "" {
					t.Fatal(errors.New("no cluster IP available"))
				}

				disabledClusterIPs = append(disabledClusterIPs, disabledClusterIP)
				toggledClusterIPs = append(toggledClusterIPs, toggledClusterIP)

				// label the disabled service
				mustOrFatal(serviceDisabled.SetLabel(labelKey, labelValue), t)
			}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			for _, services := range []kubernetes.Services{disabledServices, toggledServices} {
				if err := services.Delete(); err != nil {
					t.Fatal(err)
				}
			}
			return ctx
		}).
		Assess("should implement service.kubernetes.io/service-proxy-name", func(_ context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("verify the toggledServices are up.")
			mustOrFatal(podsUseClusterIPs(pods, toggledClusterIPs), t)
			matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: matrix.NewReachability(pods, true),
				ServiceType: entities.ClusterIP,
			}, false)

			ma.Logger.Info("verify the disabledServices are not up.")
			mustOrFatal(podsUseClusterIPs(pods, disabledClusterIPs), t)
			matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: matrix.NewReachability(pods, false),
				ServiceType: entities.ClusterIP,
			}, false)

			ma.Logger.Info("add service-proxy-name label.")
			mustOrFatal(servicesSetLabel(toggledServices, labelKey, labelValue), t)

			ma.Logger.Info("verify the toggledServices are not up.")
			mustOrFatal(podsUseClusterIPs(pods, toggledClusterIPs), t)
			matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: matrix.NewReachability(pods, false),
				ServiceType: entities.ClusterIP,
			}, false)

			ma.Logger.Info("remove service-proxy-name label.")
			mustOrFatal(servicesRemoveLabel(toggledServices, labelKey), t)

			ma.Logger.Info("verify the toggledServices are up again.")
			matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: matrix.NewReachability(pods, true),
				ServiceType: entities.ClusterIP,
			}, false)

			ma.Logger.Info("verify the disabledServices are still not up.")
			mustOrFatal(podsUseClusterIPs(pods, disabledClusterIPs), t)
			matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: matrix.NewReachability(pods, false),
				ServiceType: entities.ClusterIP,
			}, false)
			return ctx
		}).
		Feature()
	testenv.Test(t, featureProxyName)
}
