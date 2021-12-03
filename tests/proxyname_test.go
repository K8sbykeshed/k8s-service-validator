package tests

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

func mustNoWrong(wrongNum int, t *testing.T) {
	if wrongNum > 0 {
		t.Errorf("Wrong result number %d", wrongNum)
	}
}

func TestProxyNameLabel(t *testing.T) { // nolint
	pods := model.AllPods()

	// the pod where services under test is deployed
	toPod := pods[0]

	var disabledService kubernetes.ServiceBase
	var toggledService kubernetes.ServiceBase

	var disabledClusterIP string
	var toggledClusterIP string

	labelKey := "service.kubernetes.io/service-proxy-name"
	labelValue := "foo-bar"

	upReachability := matrix.NewReachability(pods, true)
	downReachability := matrix.NewReachability(pods, true)
	downReachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: toPod.Name}, false)

	featureProxyNameLabel := features.New("ProxyNameLabel").WithLabel("type", "ProxyNameLabel").
		Setup(func(_ context.Context, t *testing.T, _ *envconf.Config) context.Context {
			var err error

			// create a disabled service labeled with service-proxy-name
			disabledService = kubernetes.NewService(cs, toPod.ClusterIPService())
			if _, err := disabledService.Create(); err != nil {
				t.Fatal()
			}

			// create a service without the label, but will be toggled later
			toggledService = kubernetes.NewService(cs, toPod.ClusterIPService())
			if _, err := toggledService.Create(); err != nil {
				t.Fatal()
			}

			// wait for the disabled service
			if result, err := disabledService.WaitForEndpoint(); err != nil || !result {
				t.Fatal(errors.New("no endpoint available"))
			}

			if disabledClusterIP, err = disabledService.WaitForClusterIP(); err != nil || disabledClusterIP == "" {
				t.Fatal(errors.New("no cluster IP available"))
			}

			// wait for the toggled service
			if result, err := toggledService.WaitForEndpoint(); err != nil || !result {
				t.Fatal(errors.New("no endpoint available"))
			}

			if toggledClusterIP, err = toggledService.WaitForClusterIP(); err != nil || toggledClusterIP == "" {
				t.Fatal(errors.New("no cluster IP available"))
			}

			// label the disabled service
			mustOrFatal(disabledService.SetLabel(labelKey, labelValue), t)
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			for _, service := range []kubernetes.ServiceBase{disabledService, toggledService} {
				if err := service.Delete(); err != nil {
					t.Fatal(err)
				}
			}
			return ctx
		}).
		Assess("should implement service.kubernetes.io/service-proxy-name", func(_ context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("verify the toggledServices are up.")
			toPod.SetClusterIP(toggledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("verify the disabledServices are not up.")
			toPod.SetClusterIP(disabledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("add service-proxy-name label.")
			mustOrFatal(toggledService.SetLabel(labelKey, labelValue), t)

			ma.Logger.Info("verify the toggledServices are not up.")
			toPod.SetClusterIP(toggledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("remove service-proxy-name label.")
			mustOrFatal(toggledService.RemoveLabel(labelKey), t)

			ma.Logger.Info("verify the toggledServices are up again.")
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("verify the disabledServices are still not up.")
			toPod.SetClusterIP(disabledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)
			return ctx
		}).
		Feature()

	testenv.Test(t, featureProxyNameLabel)
}
