package suites

import (
	"context"
	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
	"github.com/k8sbykeshed/k8s-service-lb-validator/objects/data"
	"github.com/k8sbykeshed/k8s-service-lb-validator/objects/kubernetes"
	v1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

const (
	domain = "example.com"
)

// TestClusterIP hits the cluster ip on this service
func TestClusterIP(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	clusterIPFeature := features.New("Cluster IP").
		WithLabel("type", "cluster_ip").
		Assess("the cluster ip should be reachable.", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			reachability := matrix.NewReachability(pods, true)
			testCase := matrix.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: data.ClusterIP}
			wrong := matrix.ValidateOrFail(ma, model, &testCase, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		})

	env.New().BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			ma.Logger.Info("Creating a new cluster IP service batch.")
			for _, pod := range pods {
				clusterSvc := pod.ClusterIPService()

				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase
				service = kubernetes.NewService(cs, clusterSvc)
				if _, err := service.Create(); err != nil {
					log.Fatal(err)
				}

				// Wait for final status
				service.WaitForEndpoint()
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx, nil
		},
	).AfterEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) { return ctx, services.Delete() },
	).Test(t, clusterIPFeature.Feature())

}

// TestNodePort tests the existent node port and hits the node and high port allocated by this service.
func TestNodePort(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	nodePortFeature := features.New("Node Port").
		WithLabel("type", "node_port").
		Assess("the host should reachable on node port", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			reachability := matrix.NewReachability(pods, true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: data.NodePort,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		})

	env.New().BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			ma.Logger.Info("Creating a new NodePort service.")
			for _, pod := range pods {
				clusterSvc := pod.NodePortService()

				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase
				service = kubernetes.NewService(cs, clusterSvc)
				if _, err := service.Create(); err != nil {
					log.Fatal(err)
				}

				// Wait for final status
				service.WaitForEndpoint()
				nodePort, err := service.WaitForNodePort()
				if err != nil {
					log.Fatal(err)
				}

				// Set pod specification on data model
				pod.SetToPort(nodePort)
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx, nil
		},
	).AfterEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) { return ctx, services.Delete() },
	).Test(t, nodePortFeature.Feature())
}

// TestNodePortTrafficLocal test the same NodePort service (from pod-1), having ingress from different nodes
//
//			-		x-35558/pod-1	x-35558/pod-2	x-35558/pod-3	x-35558/pod-4
//	x-35558/pod-1		.				X				X				X
//	x-35558/pod-2		.				.				X				X
//	x-35558/pod-3		.				X				.				X
//	x-35558/pod-4		.				X				X				.
//
func TestNodePortTrafficLocal(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	nodeLocalFeature := features.New("NodePort Traffic Local").
		WithLabel("type", "node_port_traffic_local").
		Assess("ExternalTrafficPolicy=local", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			reachability := matrix.NewReachability(pods, false)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: data.NodePort,
			}, true)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		})

	env.New().BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			// Create a node port traffic local service for pod-1 only
			// and share the NodePort with all other pods, the test is using
			// the same port via different nodes IPs (where each pod is scheduled)
			var service kubernetes.ServiceBase

			// Create a kubernetes service based in the service spec
			service = kubernetes.NewService(cs, pods[0].NodePortLocalService())
			if _, err := service.Create(); err != nil {
				log.Fatal(err)
			}

			// Wait for final status
			service.WaitForEndpoint()
			nodePort, err := service.WaitForNodePort()
			if err != nil {
				log.Fatal(err)
			}

			// Set pod specification on data model
			for _, pod := range pods {
				pod.SetToPort(nodePort)
			}
			services = append(services, service.(*kubernetes.Service))
			return ctx, nil
		},
	).AfterEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) { return ctx, services.Delete() },
	).Test(t, nodeLocalFeature.Feature())
}

// TestLoadBalancer tests an external load balancer service created for each pod.
func TestLoadBalancer(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	loadBalancerFeature := features.New("Load Balancer").
		WithLabel("type", "load_balancer").
		Assess("load balancer should be reachable via external ip", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			reachability := matrix.NewReachability(pods, true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: data.LoadBalancer,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		},
		)

	env.New().BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			for _, pod := range pods {
				var (
					err     error
					ips     []string
					service kubernetes.ServiceBase
				)
				// Create a kubernetes service based in the service spec
				service = kubernetes.NewService(cs, pod.LoadBalancerService())
				if _, err := service.Create(); err != nil {
					return ctx, err
				}

				// Wait for final status
				service.WaitForEndpoint()

				ips, err = service.WaitForExternalIP()
				if err != nil {
					return ctx, err
				}

				// Set pod specification on data model
				pod.SetToPort(80)
				for _, ip := range ips {
					pod.SetExternalIP(ip)
				}
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx, nil
		},
	).AfterEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) { return ctx, services.Delete() },
	).Test(t, loadBalancerFeature.Feature())
}

// TestExternalService runs an External service CNAME and probes the local service on it.
func TestExternalService(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	externalSvcFeature := features.New("External Service").
		Assess("the external DNS should be reachable via local service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			reachability := matrix.NewReachability(model.AllPods(), true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: data.ClusterIP,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		},
		)

	env.New().BeforeEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
			ma.Logger.Info("Creating a new external name service batch.")
			for _, pod := range pods {
				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase
				service = kubernetes.NewService(cs, pod.ExternalNameService(domain))
				if _, err := service.Create(); err != nil {
					return ctx, err
				}

				// Wait for final status
				if _, err := service.WaitForExternalIP(); err != nil {
					return ctx, err
				}
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx, nil
		},
	).AfterEachTest(
		func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) { return ctx, services.Delete() },
	).Test(t, externalSvcFeature.Feature())
}
