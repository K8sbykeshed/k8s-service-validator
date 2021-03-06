package tests

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/k8sbykeshed/k8s-service-validator/pkg/entities"
	"github.com/k8sbykeshed/k8s-service-validator/pkg/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-validator/pkg/matrix"
	"github.com/k8sbykeshed/k8s-service-validator/pkg/tools"
)

func mustOrFatal(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestLabels(t *testing.T) { // nolint
	pods := model.AllPods()

	// the pod where services under test is deployed
	toPod := pods[0]

	var toggledService kubernetes.ServiceBase

	proxyNameLabelKey := "service.kubernetes.io/service-proxy-name"
	proxyNameLabelValue := "foo-bar"

	headlessLabelKey := "service.kubernetes.io/headless"
	headlessLabelValue := ""

	upReachability := matrix.NewReachability(pods, true)
	downReachability := matrix.NewReachability(pods, true)
	downReachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: toPod.Name}, false)

	setup := func(_ context.Context, t *testing.T, _ *envconf.Config) context.Context {
		// create a service without the label, but will be toggled later
		toggledService = kubernetes.NewService(manager.GetClientSet(), toPod.ClusterIPService())
		if _, err := toggledService.Create(); err != nil {
			t.Fatal()
		}

		// wait for the toggled service
		if result, err := toggledService.WaitForEndpoint(); err != nil || !result {
			t.Fatal(errors.New("no endpoint available"))
		}

		if toggledClusterIP, err := toggledService.WaitForClusterIP(); err != nil || toggledClusterIP == "" {
			t.Fatal(errors.New("no cluster IP available"))
		}
		return ctx
	}

	teardown := func(context.Context, *testing.T, *envconf.Config) context.Context {
		for _, service := range []kubernetes.ServiceBase{toggledService} {
			if err := service.Delete(); err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}

	getAssessFunc := func(labelKey, labelValue string) features.Func {
		return func(_ context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			zap.L().Info("verify the toggledService is up without", zap.String("label", labelKey))
			toPod.SetClusterIP(toggledService.GetClusterIP())
			tools.MustNoWrong(matrix.ValidateOrFail(manager, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false, false), t)

			zap.L().Info("add label to the toggledService", zap.String("label", labelKey))
			mustOrFatal(toggledService.SetLabel(labelKey, labelValue), t)

			zap.L().Info("verify the toggledService is not up with", zap.String("label", labelKey))
			toPod.SetClusterIP(toggledService.GetClusterIP())
			tools.MustNoWrong(matrix.ValidateOrFail(manager, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: downReachability,
				ServiceType: entities.ClusterIP,
			}, false, false), t)

			zap.L().Info("remove label from the toggledService", zap.String("label", labelKey))
			mustOrFatal(toggledService.RemoveLabel(labelKey), t)

			zap.L().Info("verify the toggledService is up again without", zap.String("label", labelKey))
			clusterIP, err := toggledService.WaitForClusterIP()
			if err != nil {
				t.Fatal(err)
			}
			toPod.SetClusterIP(clusterIP)
			tools.MustNoWrong(matrix.ValidateOrFail(manager, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: upReachability,
				ServiceType: entities.ClusterIP,
			}, false, false), t)
			return ctx
		}
	}

	featureProxyNameLabel := features.New("ProxyNameLabel").WithLabel("type", "ProxyNameLabel").
		Setup(setup).Teardown(teardown).
		Assess("should implement service.kubernetes.io/service-proxy-name",
			getAssessFunc(proxyNameLabelKey, proxyNameLabelValue)).Feature()

	featureHeadlessLabel := features.New("HeadlessLabel").WithLabel("type", "HeadlessLabel").
		Setup(setup).Teardown(teardown).
		Assess("should implement service.kubernetes.io/headless",
			getAssessFunc(headlessLabelKey, headlessLabelValue)).Feature()

	testenv.Test(t, featureProxyNameLabel, featureHeadlessLabel)
}
