package suites

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/k8sbykeshed/k8s-service-lb-validator/manager"
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/workload"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// TestClusterIP is the first test ~ it probes from exactly 3 pods to 3 cluster IPs
func TestClusterIP(t *testing.T) {
	clusterIPEnv := env.New()
	services, pods := []*v1.Service{}, model.AllPods()

	// Create clusterip service.
	clusterIPEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new cluster IP service.")
		for _, pod := range pods {
			service, err := ma.CreateService(pod.ClusterIPService())
			if err != nil {
				log.Fatal(err)
			}
			services = append(services, service)
		}
		// todo(knabben) replace this for wait and service readiness, this can take a few seconds
		// for kube-proxy set up.
		time.Sleep(4 * time.Second)
		return ctx, nil
	})

	// Execute tests.
	feature := features.New("Cluster IP").
		Assess("the cluster ip should be reachable.", func(ctx context.Context, t *testing.T) context.Context {
			// create a new matrix of reachability and test it for cluster ip
			reachability := manager.NewReachability(pods, true)
			testCase := manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: workload.ClusterIP}
			manager.ValidateOrFail(ma, model, &testCase)
			return ctx
		}).Feature()

	// Cleanup clusterip service.
	clusterIPEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		if err := ma.DeleteServices(services); err != nil {
			ma.Logger.Warn("Cant delete the service", zap.Error(err))
		}
		return ctx, nil
	})
	clusterIPEnv.Test(ctx, t, feature)
}

// TestNodePort is the second test ~ it currently probes from exactly 3 pods to the 3 nodeport backends
func TestNodePort(t *testing.T) {
	nodePortEnv, pods := env.New(), model.AllPods()
	services := []*v1.Service{}

	// Create NodePort service.
	nodePortEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new NodePort service.")
		for _, pod := range pods {
			service, err := ma.CreateService(pod.NodePortService())
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(4 * time.Second) // give some time to fw rules setup
			pod.SetToPort(service.Spec.Ports[0].NodePort)
			services = append(services, service)
		}
		return ctx, nil
	})

	// Execute tests.
	feature := features.New("Node Port").
		Assess("the host should reachable on node port", func(ctx context.Context, t *testing.T) context.Context {
			reachability := manager.NewReachability(pods, true)
			wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: workload.NodePort})
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	// Cleanup NodePort service.
	nodePortEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		if err := ma.DeleteServices(services); err != nil {
			ma.Logger.Warn("Cant delete the service", zap.Error(err))
		}
		return ctx, nil
	})
	nodePortEnv.Test(ctx, t, feature)
}

// TestLoadBalancer is the third test
func TestLoadBalancer(t *testing.T) {
	loadBalancerEnv, pods := env.New(), model.AllPods()
	services := []*v1.Service{}

	loadBalancerEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		for _, pod := range pods {
			var ips []string
			service, err := ma.CreateService(pod.LoadBalancerService())
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(4 * time.Second) // give some time to fw rules setup
			pod.SetToPort(80)
			if ips, err = ma.GetLoadBalancerService(service); err != nil {
				log.Fatal(err)
			}
			for _, ip := range ips {
				pod.SetExternalIP(ip)
			}
			services = append(services, service)
		}
		return ctx, nil
	})

	feature := features.New("Load Balancer").
		WithLabel("type", "load_balancer").
		Assess("load balancer should be reachable via external ip", func(ctx context.Context, t *testing.T) context.Context {
			reachability := manager.NewReachability(pods, true)
			wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: workload.LoadBalancer})
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	loadBalancerEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		if err := ma.DeleteServices(services); err != nil {
			ma.Logger.Warn("Cant delete the service", zap.Error(err))
		}
		return ctx, nil
	})
	loadBalancerEnv.Test(ctx, t, feature)
}

// TestExternalService
func TestExternalService(t *testing.T) {
	domain := "example.com"
	externalEnv, pods := env.New(), model.AllPods()
	services := []*v1.Service{}

	externalEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new external name service.")
		for _, pod := range pods {
			// Creating a new CNAME for domain from the local service.
			service, err := ma.CreateService(pod.ExternalNameService(domain))
			if err != nil {
				log.Fatal(err)
			}
			services = append(services, service)
		}
		time.Sleep(10 * time.Second) // give some time to fw rules setup ?
		return ctx, nil
	})

	feature := features.New("External Service").
		Assess("the external DNS should be reachable via local service", func(ctx context.Context, t *testing.T) context.Context {
			reachability := manager.NewReachability(model.AllPods(), true)
			wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability, ServiceType: workload.ClusterIP})
			if wrong > 0 {
				t.Error("Wrong result number ")
			}
			return ctx
		}).Feature()

	externalEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("after external service running")
		return ctx, nil
	})
	externalEnv.Test(ctx, t, feature)
}
