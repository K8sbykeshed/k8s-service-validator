package tests

import (
	"context"
	"errors"
	"go.uber.org/zap"
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

func TestLabels(t *testing.T) { // nolint
	pods := model.AllPods()

	// the pod where services under test is deployed
	toPod := pods[0]

	var disabledService kubernetes.ServiceBase
	var toggledService kubernetes.ServiceBase

	var disabledClusterIP string
	var toggledClusterIP string

	proxyNameLabelKey := "service.kubernetes.io/service-proxy-name"
	headlessLabelKey := "service.kubernetes.io/headless"

	labelValue := "foo-bar"

	upReachability := matrix.NewReachability(pods, true)
	downReachability := matrix.NewReachability(pods, true)
	downReachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: toPod.Name}, false)

	getSetupFunc := func(labelKey string) features.Func {
		return func(_ context.Context, t *testing.T, _ *envconf.Config) context.Context {
			var err error

			// create a disabled service with the label
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
		}
	}

	teardown := func(context.Context, *testing.T, *envconf.Config) context.Context {
		for _, service := range []kubernetes.ServiceBase{disabledService, toggledService} {
			if err := service.Delete(); err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}

	getAssessFunc := func(labelKey string) features.Func {
		return func(_ context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("verify the toggledService is up without", zap.String("label", labelKey))
			toPod.SetClusterIP(toggledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("verify the disabledService is not up with", zap.String("label", labelKey))
			toPod.SetClusterIP(disabledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("add label to the toggledService", zap.String("label", labelKey))
			mustOrFatal(toggledService.SetLabel(proxyNameLabelKey, labelValue), t)

			ma.Logger.Info("verify the toggledService is not up with", zap.String("label", labelKey))
			toPod.SetClusterIP(toggledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("remove label from the toggledService", zap.String("label", labelKey))
			mustOrFatal(toggledService.RemoveLabel(proxyNameLabelKey), t)

			ma.Logger.Info("verify the toggledService is up again without", zap.String("label", labelKey))
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)

			ma.Logger.Info("verify the disabledService is still not up with", zap.String("label", labelKey))
			toPod.SetClusterIP(disabledClusterIP)
			mustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false), t)
			return ctx
		}
	}

	featureProxyNameLabel := features.New("ProxyNameLabel").WithLabel("type", "ProxyNameLabel").
		Setup(getSetupFunc(proxyNameLabelKey)).Teardown(teardown).
		Assess("should implement service.kubernetes.io/service-proxy-name", getAssessFunc(proxyNameLabelKey)).Feature()

	featureHeadlessLabel := features.New("HeadlessLabel").WithLabel("type", "HeadlessLabel").
		Setup(getSetupFunc(headlessLabelKey)).Teardown(teardown).
		Assess("should implement service.kubernetes.io/headless", getAssessFunc(headlessLabelKey)).Feature()

	testenv.Test(t, featureProxyNameLabel, featureHeadlessLabel)
}
