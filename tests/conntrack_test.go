package suites

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	v1 "k8s.io/api/core/v1"
)

var (
	udpPort          int32 = 80
	udpNamespaceName string
)

// podUDPStaleServer returns server pod spec
func podUDPStaleServer(podName string, node *v1.Node) (*entities.Pod, error) {
	pod := &entities.Pod{
		Name:      podName,
		Namespace: udpNamespaceName,
		InitContainers: []*entities.Container{{
			Command: []string{"/bin/sh", "-c", "echo Pausing start. && sleep 10"},
		}},
		Containers: []*entities.Container{
			{Port: udpPort, Protocol: v1.ProtocolUDP},
		},
		NodeName: node.Name,
	}

	if _, err := ma.CreatePod(pod.ToK8SSpec()); err != nil {
		return nil, err
	}
	if err := ma.WaitAndSetIPs(pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// podUDPStaleClient returns client pod spec
func podUDPStaleClient(podName, clusterIP string, node *v1.Node) (*entities.Pod, error) {
	cmd := fmt.Sprintf(`date; for i in $(seq 1 3000); do echo "$(date) Try: ${i}"; echo hostname | nc -u -w2 %s %d; echo; done`, clusterIP, udpPort)
	pod := &entities.Pod{
		SkipProbe: true,
		Name:      podName,
		NodeName:  node.Name,
		Namespace: udpNamespaceName,
		Containers: []*entities.Container{
			{Protocol: v1.ProtocolUDP, Port: 80},
			{Image: "busybox", Command: []string{"/bin/sh", "-c", cmd}},
		},
	}

	if _, err := ma.CreatePod(pod.ToK8SSpec()); err != nil {
		return nil, err
	}
	if err := ma.WaitAndSetIPs(pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// 1. Create an UDP Service
// 2. Client Pod sending traffic to the UDP service
// 3. Create an UDP server associated to the Service created in 1. with an init container that sleeps for some time
// The init container makes that the server pod is not ready, however, the endpoint slices are created, it is just
// that the Endpoint conditions Ready is false.
func TestUDPInitContainer(t *testing.T) {
	var (
		udpModel matrix.Model
		services kubernetes.Services
	)

	featureUDPInitContainer := features.New("UDP stale endpoint").WithLabel("type", "udp_stale_endpoint").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			var (
				err                 error
				result              bool
				scheduledNodes      []*v1.Node
				firstPod, secondPod *entities.Pod
			)

			// create and start namespace and ready nodes
			udpNamespaceName = matrix.GetNamespace()
			udpNamespace := entities.Namespace{Name: udpNamespaceName}
			if _, err := ma.CreateNamespace(udpNamespace.Spec()); err != nil {
				t.Fatal(err)
			}
			if scheduledNodes, err = ma.GetReadyNodes(); err != nil {
				t.Fatal(err)
			}

			// create stale server pod
			if firstPod, err = podUDPStaleServer("pod-1", scheduledNodes[0]); err != nil {
				t.Fatal(err)
			}

			// create a cluster service based on backend server
			clusterSvc := firstPod.ClusterIPService()
			var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
			if _, err := service.Create(); err != nil {
				t.Fatal(err)
			}
			var clusterIP string
			// wait for final status
			if result, err = service.WaitForEndpoint(); err != nil || !result {
				t.Fatal(errors.New("no endpoint available"))
			}
			if clusterIP, err = service.WaitForClusterIP(); err != nil || clusterIP == "" {
				t.Fatal(errors.New("no cluster IP available"))
			}

			firstPod.SetClusterIP(clusterIP)
			services = append(services, service.(*kubernetes.Service))

			// Create a pod in one node to create the UDP traffic against the ClusterIP service every 5 seconds
			// start a stale connection without marking stale creates a wrong conn track entry
			// invalidating the NAT cache
			if secondPod, err = podUDPStaleClient("pod-2", clusterIP, scheduledNodes[1]); err != nil {
				t.Fatal(err)
			}

			// start the model with the namespace
			udpNamespace.Pods = []*entities.Pod{firstPod, secondPod}
			udpModel = *matrix.NewModelWithNamespace([]*entities.Namespace{&udpNamespace}, dnsDomain)

			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			logger.Info("Cleanup namespace.")
			if err := ma.DeleteNamespaces([]string{udpNamespaceName}); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating UDP and conntrack entries")
			reachability := matrix.NewReachability(udpModel.AllPods(), true)

			// Note that the fact that Endpoints object already exists, does NOT mean
			// that iptables (or whatever else is used) was already programmed.
			// Additionally take into account that UDP conntract entries timeout is
			// 30 seconds by default.
			// Based on the above check if the pod receives the traffic.

			testCase := matrix.TestCase{ToPort: 80, Protocol: v1.ProtocolUDP, Reachability: reachability, ServiceType: entities.ClusterIP}
			wrong := matrix.ValidateOrFail(ma, &udpModel, &testCase, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	testenv.Test(t, featureUDPInitContainer)
}
