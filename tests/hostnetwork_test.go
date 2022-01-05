package tests

import (
	"context"
	"testing"

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	"github.com/k8sbykeshed/k8s-service-validator/matrix"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestHostNetwork(t *testing.T) {
	var newPod *entities.Pod
	// 1. Create new pod-5 using hostNetwork in the existing namespace on node-1
	// 2. verify successful connection between pod-5 and all pods in the cluster
	testHostNetwork := features.New("HostNetwork").WithLabel("type", "hostNetwork").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			nodes, err := manager.GetReadyNodes()
			if err != nil {
				t.Fatal(err)
			}

			newPod, err = createHostNetworkPod("pod-5", nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			model.AddPod(newPod, namespace)

			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			zap.L().Info("delete newly created pod, which use host network.")
			if err := manager.DeletePod(newPod.Name, newPod.Namespace); err != nil {
				t.Fatal(err)
			}
			err := model.RemovePod(newPod.Name, namespace)
			if err != nil {
				zap.L().Debug(err.Error())
			}
			return ctx
		}).
		Assess("should function for pods using hostNetwork", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			zap.L().Info("testing pod with hostNetwork connections.")
			// Expect pod-5 can connect with pods in the cluster
			reachability := matrix.NewReachability(model.AllPods(), true)

			testCase := matrix.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.PodIP}
			wrong := matrix.ValidateOrFail(manager, model, &testCase, false, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	testenv.Test(t, testHostNetwork)
}

func createHostNetworkPod(podName string, node *v1.Node) (*entities.Pod, error) {
	pod := &entities.Pod{
		Name:        podName,
		Namespace:   namespace,
		HostNetwork: true,

		Containers: []*entities.Container{
			{Port: 80, Protocol: v1.ProtocolTCP},
		},
		NodeName: node.Name,
	}

	if _, err := manager.CreatePod(pod.ToK8SSpec()); err != nil {
		return nil, err
	}
	if err := manager.WaitAndSetIPs(pod); err != nil {
		return nil, err
	}

	return pod, nil
}
