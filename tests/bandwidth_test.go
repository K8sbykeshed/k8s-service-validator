package tests

import (
	"context"
	"fmt"
	"github.com/k8sbykeshed/k8s-service-validator/consts"
	"log"
	"testing"

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	"github.com/k8sbykeshed/k8s-service-validator/matrix"
	"github.com/k8sbykeshed/k8s-service-validator/tools"
	"go.uber.org/zap"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// 1. Create a new namespace e.g. x-70212-iperf
// 2. For each node, launch an iperf server pod listening to port 80 (TCP)
// 3. Probe reachability and measure from pod-A to pod-B
func TestBandwidth(t *testing.T) {
	var (
		iperfPodNames      []string
		iperfNamespaceName string
		err                error

		nodes          []*v1.Node
		pods           []*entities.Pod
		iperfNamespace *entities.Namespace
	)

	featureBandwidth := features.New("Bandwidth among nodes").WithLabel("type", "iperf").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			iperfNamespaceName = matrix.GetIPerfNamespace()
			if nodes, err = manager.GetReadyNodes(); err != nil {
				log.Fatal(err)
			}
			zap.L().Info("Deploy iperf servers for each node in namespace", zap.String("namespace", iperfNamespaceName))
			// Generate pod names using existing nodes as iperf servers
			for i := 1; i <= len(nodes); i++ {
				iperfPodNames = append(iperfPodNames, fmt.Sprintf("pod-%d-iperf", i))
			}
			iperfNamespace = model.AddIPerfNamespace(iperfNamespaceName, iperfPodNames, []int32{80}, []v1.Protocol{v1.ProtocolTCP})
			if err = manager.StartPodsInNamespace(model, nodes, iperfNamespace); err != nil {
				log.Fatal(err)
			}
			zap.L().Info("Wait and set iPerf pods IP.")
			pods = model.AllIPerfPods()
			for _, pod := range pods {
				if err = manager.WaitAndSetIPs(pod); err != nil {
					log.Fatal(err)
				}
			}
			if len(manager.PendingPods) > 0 {
				// Remove pods which are pending because of taints
				zap.L().Info(fmt.Sprintf("Removing %v iperf pods as stale in pending(likely because of taints).", len(manager.PendingPods)))
				for pendingPod, pollTimes := range manager.PendingPods {
					if pollTimes > consts.PollTimesToDeterminePendingPod {
						err := model.RemovePod(pendingPod, iperfNamespaceName)
						if err != nil {
							zap.L().Debug(err.Error())
						}
						if err := manager.DeletePod(pendingPod, iperfNamespaceName); err != nil {
							log.Fatal(err)
						}
						delete(manager.PendingPods, pendingPod)
					}
				}
			}
			// Wait until HTTP servers including iperf are up.
			if err = manager.WaitForHTTPServers(model); err != nil {
				log.Fatal(err)
			}
			return ctx
		}).
		Assess("bandwidth test",
			func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
				pods = model.AllIPerfPods()

				zap.L().Info("Measure bandwidth across nodes.")
				reachabilityTCP := matrix.NewReachability(pods, true)
				tools.MustNoWrong(matrix.ValidateAndMeasureBandwidthOrFail(manager, model, &matrix.TestCase{
					ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.PodIP,
				}, false, false, true), t)
				return ctx
			}).
		Teardown(
			func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
				zap.L().Info("Cleanup iperf namespace.", zap.String("namespace", iperfNamespaceName))
				if err := manager.DeleteNamespaces([]string{iperfNamespaceName}); err != nil {
					log.Fatal(err)
				}
				return ctx
			}).
		Feature()

	testenv.Test(t, featureBandwidth)
}
