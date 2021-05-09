package suites

import (
	"context"
	"github.com/K8sbykeshed/svc-tests/manager"
	v1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func TestClusterIP(t *testing.T) {
	cs, config := clientSet()
	feat := features.New("Cluster IP").
		Assess("pods are reachable", func(ctx context.Context, t *testing.T) context.Context {
			nsX, model, ma := manager.GetModel(cs, config)

			for _, pod := range model.AllPods() {
				if pod.PodString() == "name-x/a" {
					if _, err := ma.CreateService(pod.Service()); err != nil {
						log.Fatal(err)
					}
				}
			}

			reachability := manager.NewReachability(model.AllPods(), false)
			reachability.ExpectPeer(&manager.Peer{Namespace: nsX}, &manager.Peer{Namespace: nsX, Pod: "a"}, true)
			manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability})

			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
