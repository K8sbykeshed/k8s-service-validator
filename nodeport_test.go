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

const nodePort = 31010

func createNodePortService() {
	for _, pod := range model.AllPods() {
		if string(pod.PodString()) == fmt.Sprintf("%s/a", namespace) {
			if _, err := ma.CreateService(pod.NodePortService(nodePort)); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func TestNodePort(t *testing.T) {
	feat := features.New("Node port").
		Assess("host is reachable by node port", func(ctx context.Context, t *testing.T) context.Context {
			createNodePortService()

			reachability := manager.NewReachability(model.AllPods(), false)
			reachability.ExpectPeer(&manager.Peer{Namespace: namespace}, &manager.Peer{Namespace: namespace, Pod: "a"}, true)
			manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: nodePort, Protocol: v1.ProtocolTCP, Reachability: reachability})

			return ctx
		}).Feature()
	testenv.Test(ctx, t, feat)
}
