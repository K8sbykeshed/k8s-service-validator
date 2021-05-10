package suites

import (
	"context"
	"github.com/k8sbykeshed/svc-tests/manager"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"
)

var (
	waitInterval = 8 * time.Second
)

func createNodePortService() []*v1.Service {
	pods := model.AllPods()
	services := make([]*v1.Service, len(pods))
	for i, pod := range pods {
		service, err := ma.CreateService(pod.NodePortService())
		if err != nil {
			log.Fatal(err)
		}
		services[i] = service
	}
	time.Sleep(waitInterval)  // give some time to fw rules setup
	return services
}

func createClusterIPService() {
	for _, pod := range model.AllPods() {
		if _, err := ma.CreateService(pod.ClusterIPService()); err != nil {
			log.Fatal(err)
		}
	}
}

func TestClusterIP(t *testing.T) {
	feat := features.New("Cluster IP").
		Assess("pods are reachable", func(ctx context.Context, t *testing.T) context.Context {
			createClusterIPService()
			reachability := manager.NewReachability(model.AllPods(), true)
			manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability})
			return ctx
		}).Feature()

	testenv.Test(ctx, t, feat)
}

func TestNodePort(t *testing.T) {
	feat := features.New("Node port").
		Assess("host is reachable by node port", func(ctx context.Context, t *testing.T) context.Context {
			for _, service := range createNodePortService() {
				for _, port := range service.Spec.Ports {
					ma.Logger.Info("Evaluating node port.", zap.Int32("nodeport", port.NodePort))
					reachability := manager.NewReachability(model.AllPods(), true)
					wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: int(port.NodePort), Protocol: v1.ProtocolTCP, Reachability: reachability})
					if wrong > 0 {
						t.Error("Wrong result number ")
					}
				}
			}
			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
