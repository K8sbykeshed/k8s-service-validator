package suites

import (
	"context"
	"fmt"
	"github.com/K8sbykeshed/svc-tests/manager"
	v1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func createService() {
	for _, pod := range model.AllPods() {
		if string(pod.PodString()) == fmt.Sprintf("%s/a", namespace) {
			if _, err := ma.CreateService(pod.Service()); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func TestClusterIP(t *testing.T) {
	feat := features.New("Cluster IP").
		Assess("pods are reachable", func(ctx context.Context, t *testing.T) context.Context {
			createService()

			reachability := manager.NewReachability(model.AllPods(), false)
			reachability.ExpectPeer(&manager.Peer{Namespace: namespace}, &manager.Peer{Namespace: namespace, Pod: "a"}, true)
			manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability})
			// todo(knabben) - validate and assert results

			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
