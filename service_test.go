package suites

import (
	"context"
	"errors"
	"fmt"
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

// TestBasicService starts up the basic Kubernetes services available
func TestBasicService(t *testing.T) {
	pods := model.AllPods()
	var services kubernetes.Services

	featureClusterIP := features.New("ClusterIP").WithLabel("type", "cluster_ip").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			for _, pod := range pods {
				clusterSvc := pod.ClusterIPService()

				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
				if _, err := service.Create(); err != nil {
					t.Fatal(err)
				}

				// Wait for final status
				result, err := service.WaitForEndpoint()
				if err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}
				services = append(services, service.(*kubernetes.Service))
			}
			ma.Logger.Info(fmt.Sprintf("%v",services))
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable.", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Starting ClusterIP")
			reachability := matrix.NewReachability(pods, true)
			testCase := matrix.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ClusterIP}
			wrong := matrix.ValidateOrFail(ma, model, &testCase, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	featureNodePort := features.New("NodePort").WithLabel("type", "node_port").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			for _, pod := range pods {
				clusterSvc := pod.NodePortService()

				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
				if _, err := service.Create(); err != nil {
					t.Fatal(err)
				}

				// Wait for final status
				result, err := service.WaitForEndpoint()
				if err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}
				nodePort, err := service.WaitForNodePort()
				if err != nil {
					t.Fatal(err)
				}

				// Set pod specification on model model
				pod.SetToPort(nodePort)
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should reachable on node port TCP and UDP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Starting TCP NodePort service.")
			reachabilityTCP := matrix.NewReachability(pods, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.NodePort,
			}, false)
			if wrongTCP > 0 {
				t.Error("[NodePort TCP] Wrong result number ")
			}

			t.Log("Starting UDP NodePort service.")
			reachabilityUDP := matrix.NewReachability(pods, true)
			wrongUDP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.NodePort,
			}, false)
			if wrongUDP > 0 {
				t.Error("[NodePort UDP] Wrong result number ")
			}
			return ctx
		}).Feature()

	featureLoadBalancer := features.New("LoadBalancer").WithLabel("type", "load_balancer").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			for _, pod := range pods {
				var (
					err error
					ips []entities.ExternalIP
				)
				// Create a load balancers with TCP/UDP ports, based in the service spec
				serviceTCP := kubernetes.NewService(cs, pod.LoadBalancerServiceByProtocol(v1.ProtocolTCP))
				if _, err := serviceTCP.Create(); err != nil {
					t.Fatal(err)
				}
				serviceUDP := kubernetes.NewService(cs, pod.LoadBalancerServiceByProtocol(v1.ProtocolUDP))
				if _, err := serviceUDP.Create(); err != nil {
					t.Fatal(err)
				}

				// Wait for final status
				result, err := serviceTCP.WaitForEndpoint()
				if err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}

				result, err = serviceUDP.WaitForEndpoint()
				if err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}

				ipsForTCP, err := serviceTCP.WaitForExternalIP()
				if err != nil {
					t.Fatal(err)
				}
				ips = append(ips, entities.NewExternalIPs(ipsForTCP, v1.ProtocolTCP)...)

				ipsForUDP, err := serviceUDP.WaitForExternalIP()
				if err != nil {
					t.Fatal(err)
				}
				ips = append(ips, entities.NewExternalIPs(ipsForUDP, v1.ProtocolUDP)...)

				if len(ips) == 0 {
					t.Fatal(errors.New("invalid external UDP IPs setup"))
				}

				// Set pod specification on data model
				pod.SetToPort(80)
				pod.SetExternalIPs(ips)
				services = append(services, serviceTCP)
				services = append(services, serviceUDP)
			}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable via node IP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("NodePort TCP Loadbalancer tests")
			reachabilityTCP := matrix.NewReachability(pods, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.LoadBalancer,
			}, false)
			if wrongTCP > 0 {
				t.Error("[Load Balancer TCP] Wrong result number")
			}

			t.Log("NodePort UDP Loadbalancer tests")
			reachabilityUDP := matrix.NewReachability(pods, true)
			wrongUDP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.LoadBalancer,
			}, false)
			if wrongUDP > 0 {
				t.Error("[Load Balancer UDP] Wrong result number")
			}
			return ctx
		}).Feature()

	testenv.Test(t, featureClusterIP, featureNodePort, featureLoadBalancer)
}

func TestExternalService(t *testing.T) {
	const (
		domain = "example.com"
	)

	pods := model.AllPods()
	var services kubernetes.Services

	// Create a node port traffic local service for pod-1 only
	// and share the NodePort with all other pods, the test is using
	// the same port via different nodes IPs (where each pod is scheduled)
	featureNodeLocal := features.New("NodePort Traffic Local").WithLabel("type", "node_port_traffic_local").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			// Create a kubernetes service based in the service spec
			var service kubernetes.ServiceBase = kubernetes.NewService(cs, pods[0].NodePortLocalService())
			if _, err := service.Create(); err != nil {
				t.Fatal(err)
			}

			// Wait for final status
			result, err := service.WaitForEndpoint()
			if err != nil || !result {
				t.Fatal(errors.New("no endpoint available"))
			}
			nodePort, err := service.WaitForNodePort()
			if err != nil {
				t.Fatal(err)
			}

			// Set pod specification on model model
			for _, pod := range pods {
				pod.SetToPort(nodePort)
			}
			services = append(services, service.(*kubernetes.Service))
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating ExternalTrafficPolicy=local")
			reachability := matrix.NewReachability(pods, false)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.NodePort,
			}, true)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	featureExternal := features.New("External Service").WithLabel("type", "external").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			for _, pod := range pods {
				// Create a kubernetes service based in the service spec
				var service kubernetes.ServiceBase = kubernetes.NewService(cs, pod.ExternalNameService(domain))
				if _, err := service.Create(); err != nil {
					t.Fatal(err)
				}

				// Wait for final status
				if _, err := service.WaitForExternalIP(); err != nil {
					t.Fatal(err)
				}
				services = append(services, service.(*kubernetes.Service))
			}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating External service")
			reachability := matrix.NewReachability(model.AllPods(), true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ClusterIP,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	testenv.Test(t, featureNodeLocal, featureExternal)
}
