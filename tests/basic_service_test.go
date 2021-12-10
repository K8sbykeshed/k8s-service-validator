package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	"github.com/k8sbykeshed/k8s-service-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-validator/matrix"
	"github.com/k8sbykeshed/k8s-service-validator/tools"
	"github.com/pkg/errors"
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
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should be reachable via cluster IP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing ClusterIP with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.ClusterIP,
			}, false, false), t)

			ma.Logger.Info("Testing ClusterIP with UDP protocol.")
			reachabilityUDP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.ClusterIP,
			}, false, false), t)
			return ctx
		}).Feature()

	// Test session affinity clientIP
	featureSessionAffinity := features.New("SessionAffinity").WithLabel("type", "cluster_ip_sessionAffinity").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			// add new label to two pods, pod-3 and pod-4
			labelKey := "app"
			labelValue := "test-session-affinity"
			podsWithNewLabel := pods[2:]
			for _, pod := range podsWithNewLabel {
				ma.AddLabelToPod(pod, labelKey, labelValue)
			}
			// create cluster IP service with the new label and session affinity: clientIP
			_, service, clusterIP, err := entities.CreateServiceFromTemplate(cs, entities.ServiceTemplate{
				Name:            "service-session-affinity",
				Namespace:       namespace,
				Selector:        map[string]string{labelKey: labelValue},
				SessionAffinity: true,
				ProtocolPort:    entities.ProtocolPortPair{string(pods[0].Containers[0].Protocol), 80},
			})
			if err != nil {
				t.Fatal(err)
			}

			for _, p := range pods {
				p.SetClusterIP(clusterIP)
				p.SetToPort(80)
			}

			services = []*kubernetes.Service{service.(*kubernetes.Service)}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			podsWithNewLabel := pods[3:]
			for _, pod := range podsWithNewLabel {
				ma.RemoveLabelFromPod(pod, "app")
			}
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should always reach to same pod", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing ClusterIP session affinity to one pod.")

			clusterIPWithSessionAffinity := pods[3].GetClusterIP()
			// setup affinity
			fromToPeer := map[string]string{}
			for _, p := range pods {
				connected, endpoint, connectCmd, err := ma.ProbeConnectivityWithCurl(namespace, p.Name, p.Containers[0].Name, clusterIPWithSessionAffinity, v1.ProtocolTCP, 80)
				if err != nil {
					t.Fatal(errors.Wrapf(err, "failed to establish affinity with cmd: %v", connectCmd))
				}
				if !connected {
					t.Fatal(errors.New("failed to connect the ClusterIP service with sessionAffinity."))
				}
				fromToPeer[p.Name] = endpoint
			}


			ma.Logger.Info(fmt.Sprintf("Session affinity service, from/to peers: %v", fromToPeer))
			reachability := matrix.NewReachability(pods, false)
			for from, to := range fromToPeer {
				reachability.ExpectPeer(&matrix.Peer{Namespace: namespace, Pod: from}, &matrix.Peer{Namespace: namespace, Pod: to}, true)
			}
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ClusterIP,
			}, false, true), t)

			return ctx
		}).Feature()

	// Test endless service
	featureEndlessService := features.New("EndlessService").WithLabel("type", "cluster_ip_endless").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			var endlessServicePort int32 = 80
			// Create a service with no endpoints
			_, service, clusterIP, err := entities.CreateServiceFromTemplate(cs, entities.ServiceTemplate{
				Name:         "endless",
				Namespace:    namespace,
				ProtocolPort: entities.ProtocolPortPair{string(v1.ProtocolTCP), endlessServicePort},
			})
			if err != nil {
				t.Fatal(err)
			}

			for _, pod := range pods {
				pod.SetClusterIP(clusterIP)
				pod.SetToPort(endlessServicePort)
			}

			services = []*kubernetes.Service{service.(*kubernetes.Service)}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should not be reachable via endless service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing Endless service.")
			reachability := matrix.NewReachability(model.AllPods(), true)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace}, false)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ClusterIP,
			}, false, false), t)
			return ctx
		}).Feature()

	// Test hairpin
	featureHairpin := features.New("Hairpin").WithLabel("type", "cluster_ip_hairpin").
		Setup(func(context.Context, *testing.T, *envconf.Config) context.Context {
			services = make(kubernetes.Services, len(pods))
			// Create a clusterIP service
			serviceName, service, _, err := entities.CreateServiceFromTemplate(cs, entities.ServiceTemplate{
				Name:         "hairpin",
				Namespace:    namespace,
				ProtocolPort: entities.ProtocolPortPair{string(pods[0].Containers[0].Protocol), 80},
			})
			if err != nil {
				t.Fatal(err)
			}

			pods[0].SetServiceName(serviceName)

			services = []*kubernetes.Service{service.(*kubernetes.Service)}
			return ctx
		}).
		Teardown(func(context.Context, *testing.T, *envconf.Config) context.Context {
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should be reachable for hairpin", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing hairpin.")
			reachability := matrix.NewReachability(model.AllPods(), true)
			reachability.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: pods[0].Name}, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ClusterIP,
			}, false, false), t)
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
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should reachable on node port TCP and UDP", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Testing NodePort with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.NodePort,
			}, false, false), t)

			ma.Logger.Info("Testing NodePort with UDP protocol.")
			reachabilityUDP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.NodePort,
			}, false, false), t)
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
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should be reachable via load balancer", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating load balancer with TCP protocol")
			reachabilityTCP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.LoadBalancer,
			}, false, false), t)

			ma.Logger.Info("Creating Loadbalancer with UDP protocol")
			reachabilityUDP := matrix.NewReachability(pods, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolUDP, Reachability: reachabilityUDP, ServiceType: entities.LoadBalancer,
			}, false, false), t)
			return ctx
		}).Feature()

	testenv.Test(t, featureClusterIP, featureNodePort, featureLoadBalancer, featureEndlessService, featureHairpin,
		featureSessionAffinity)
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
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should be reachable via NodePortLocal k8s service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating ExternalTrafficPolicy=locali with ports in TCP and UDP")
			ma.Logger.Info("Testing NodePortLocal with TCP protocol.")
			reachabilityTCP := matrix.NewReachability(pods, false)
			reachabilityTCP.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityTCP, ServiceType: entities.NodePort,
			}, true, false), t)

			ma.Logger.Info("Testing NodePortLocal with UDP protocol.")
			reachabilityUDP := matrix.NewReachability(pods, false)
			reachabilityUDP.ExpectPeer(&matrix.Peer{Namespace: namespace}, &matrix.Peer{Namespace: namespace, Pod: "pod-1"}, true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				Protocol: v1.ProtocolTCP, Reachability: reachabilityUDP, ServiceType: entities.NodePort,
			}, true, false), t)
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
			tools.ResetTestBoard(t, services, model)
			return ctx
		}).
		Assess("should be reachable via ExternalName k8s service", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ma.Logger.Info("Creating External service")
			reachability := matrix.NewReachability(model.AllPods(), true)
			tools.MustNoWrong(matrix.ValidateOrFail(ma, model, &matrix.TestCase{
				ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: entities.ExternalName,
			}, false, false), t)
			return ctx
		}).Feature()

	testenv.Test(t, featureNodeLocal, featureExternal)
}
