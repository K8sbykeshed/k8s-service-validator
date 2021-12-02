package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-lb-validator/matrix"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// TestBasicService starts up the basic Kubernetes services available
func TestBasicService(t *testing.T) { // nolint
	pods := model.AllPods()
	var services kubernetes.Services

	featureClusterIP := features.New("ClusterIP").WithLabel("type", "cluster_ip").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			for _, pod := range pods {
				var (
					err       error
					result    bool
					clusterIP string
				)
				// Create a kubernetes service based in the service spec
				clusterSvc := pod.ClusterIPService()
				var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
				if _, err := service.Create(); err != nil {
					t.Fatal(err)
				}

				// wait for final status
				if result, err = service.WaitForEndpoint(); err != nil || !result {
					t.Fatal(errors.New("no endpoint available"))
				}
				if clusterIP, err = service.WaitForClusterIP(); err != nil || clusterIP == "" {
					t.Fatal(errors.New("no cluster IP available"))
				}
				pod.SetClusterIP(clusterIP)
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
		Assess("should be reachable via cluster IP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing ClusterIP with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.ClusterIP,
			}, false)
			if wrongTCP > 0 {
				t.Error("Wrong result number ")
			}

			ma.Logger.Info("Testing ClusterIP with UDP protocol.")
			reachabilityUDP := matrix.NewReachability(pods, true)
			wrongUDP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachabilityUDP, ServiceType: entities.ClusterIP,
			}, false)
			if wrongUDP > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	// Test endless service
	featureEndlessService := features.New("EndlessService").WithLabel("type", "cluster_ip_endless").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			var endlessServicePort int32 = 80
			// Create a service with no endpoints
			clusterSvc := entities.NewServiceFromTemplate(entities.Service{Name: "endless", Namespace: namespace})
			clusterSvc.Spec.Ports = []v1.ServicePort{{
				Name:     fmt.Sprintf("service-port-%s-%d", strings.ToLower(string(v1.ProtocolTCP)), endlessServicePort),
				Protocol: v1.ProtocolTCP,
				Port:     endlessServicePort,
			}}
			var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
			if _, err := service.Create(); err != nil {
				t.Fatal(err)
			}

			// wait for final status
			if _, err := service.WaitForEndpoint(); err != nil {
				t.Fatal(err)
			}

			for _, pod := range pods {
				pod.SetServiceName(clusterSvc.Name)
				pod.SetToPort(endlessServicePort)
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
		Assess("should be reachable via cluster IP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing Endless service.")
			reachability := matrix.NewReachability(model.AllPods(), true)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace}, false)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ServiceName,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	// Test hairpin
	featureHairpin := features.New("Hairpin").WithLabel("type", "cluster_ip_hairpin").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			// Create a service with no endpoints
			clusterSvc := entities.NewServiceFromTemplate(entities.Service{Name: "hairpin", Namespace: namespace})
			clusterSvc.Spec.Ports = []v1.ServicePort{{
				Name:     fmt.Sprintf("service-port-%s-%d", strings.ToLower(string(pods[0].Containers[0].Protocol)), pods[0].Containers[0].Port),
				Protocol: pods[0].Containers[0].Protocol,
				Port:     pods[0].Containers[0].Port,
			}}
			clusterSvc.Spec.Selector = pods[0].LabelSelector()
			var service kubernetes.ServiceBase = kubernetes.NewService(cs, clusterSvc)
			if _, err := service.Create(); err != nil {
				t.Fatal(err)
			}

			// wait for final status
			if _, err := service.WaitForEndpoint(); err != nil {
				t.Fatal(err)
			}

			pods[0].SetServiceName(clusterSvc.Name)
			pods[0].SetToPort(pods[0].Containers[0].Port)

			services = append(services, service.(*kubernetes.Service))
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable for hairpin", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing hairpin.")
			reachability := matrix.NewReachability(model.AllPods(), true)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: pods[0].Name}, true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ServiceName,
			}, false)
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

				// Set pod specification on entity model
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
			ma.Logger.Info("Testing NodePort with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.NodePort,
			}, false)
			if wrongTCP > 0 {
				t.Error("[NodePort TCP] Wrong result number ")
			}

			ma.Logger.Info("Testing NodePort with UDP protocol.")
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

				// Set pod specification on entity model
				pod.SetToPort(80)
				pod.SetExternalIPs(ips)

				services = append(services, serviceTCP, serviceUDP)
			}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			if err := services.Delete(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("should be reachable via load balancer", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating load balancer with TCP protocol")
			reachabilityTCP := matrix.NewReachability(pods, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.LoadBalancer,
			}, false)
			if wrongTCP > 0 {
				t.Error("[Load Balancer TCP] Wrong result number")
			}

			ma.Logger.Info("Creating Loadbalancer with UDP protocol")
			reachabilityUDP := matrix.NewReachability(pods, true)
			wrongUDP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.LoadBalancer,
			}, false)
			if wrongUDP > 0 {
				t.Error("[Load Balancer UDP] Wrong result number")
			}
			return ctx
		}).Feature()

	testenv.Test(t, featureClusterIP, featureNodePort, featureLoadBalancer, featureEndlessService, featureHairpin)
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

			// Set pod specification on entity model
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
		Assess("should be reachable via NodePortLocal k8s service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating ExternalTrafficPolicy=locali with ports in TCP and UDP")
			ma.Logger.Info("Testing NodePortLocal with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, false)
			reachabilityTCP.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			wrongTCP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.NodePort,
			}, true)
			if wrongTCP > 0 {
				t.Error("Wrong result number ")
			}

			ma.Logger.Info("Testing NodePortLocal with UDP protocol.")
			reachabilityUDP := matrix.NewReachability(pods, false)
			reachabilityUDP.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			wrongUDP := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityUDP, ServiceType: entities.NodePort,
			}, true)
			if wrongUDP > 0 {
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
				k8sSvc, err := service.Create()
				if err != nil {
					t.Fatal(err)
				}

				pod.SetServiceName(k8sSvc.Name)
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
		Assess("should be reachable via ExternalName k8s service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating External service")
			reachability := matrix.NewReachability(model.AllPods(), true)
			wrong := matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ServiceName,
			}, false)
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	testenv.Test(t, featureNodeLocal, featureExternal)
}
