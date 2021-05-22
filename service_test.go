package suites

import (
	"context"
	"fmt"
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"
)

var (
	waitInterval = 8 * time.Second
)

// TestClusterIP is the first test we run ~ it probes from exactly 3 pods to 3 cluster IPs
func TestClusterIP(t *testing.T) {
	clusterIPEnv := env.New()
	services, pods := []*v1.Service{}, model.AllPods()

	// Create clusterip service.
	clusterIPEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new cluster IP service.")
		for _, pod := range pods {
			svc, err := ma.CreateService(pod.ClusterIPService())
			if err != nil {
				log.Fatal(err)
			}
			services = append(services, svc)
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
			testCase := manager.TestCase{ToPort: 80, Protocol: v1.ProtocolTCP, Reachability: reachability}
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

// TestNodePort is the second test we can run ~ it currently probes from exactly 3 pods to the 3 nodeport backends
func TestNodePort(t *testing.T) {
	nodePortEnv := env.New()
	pods := model.AllPods()
	services := make([]*v1.Service, len(pods))

	// Create NodePort service.
	nodePortEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new NodePort service.")
		for i, pod := range pods {
			service, err := ma.CreateService(pod.NodePortService())
			if err != nil {
				log.Fatal(err)
			}
			services[i] = service
		}
		time.Sleep(4 * time.Second) // give some time to fw rules setup
		return ctx, nil
	})

	// Execute tests.
	feature := features.New("Node Port").
		Assess("the host should reachable on node port", func(ctx context.Context, t *testing.T) context.Context {
			// todo(knabben) - refactor for only one matrix instead of m*n
			for _, service := range services {
				for _, port := range service.Spec.Ports {
					ma.Logger.Info("Evaluating node port.", zap.Int32("nodeport", port.NodePort))
					reachability := manager.NewReachability(pods, true)
					wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: int(port.NodePort), Protocol: v1.ProtocolTCP, Reachability: reachability})
					if wrong > 0 {
						t.Error("Wrong result number ")
					}
				}
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


// TestLoadBalancer is the second test we can run ~ it currently probes from exactly 3 pods to the 3 nodeport backends
func TestLoadBalancer(t *testing.T) {
	loadBalancerEnv := env.New()
	loadBalancerEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("before load balancer running")
		return ctx, nil
	})

	feature := features.New("Load Balancer").
		WithLabel("type", "load_balancer").
		Assess("load balancer should be reachable via external ip", func(ctx context.Context, t *testing.T) context.Context {
			ma.Logger.Info("load balancer in the middle")
			return ctx
		}).Feature()

	loadBalancerEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("after node port running")
		return ctx, nil
	})
	loadBalancerEnv.Test(ctx, t, feature)
}

// TestExternalService
func _TestExternalService(t *testing.T) {
	pods := model.AllPods()
	services := make([]*v1.Service, len(pods))

	externalEnv := env.New()
	externalEnv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		ma.Logger.Info("Creating a new External name service.")
		for i, pod := range pods {
			service, err := ma.CreateService(pod.ExternalNameService())
			if err != nil {
				log.Fatal(err)
			}
			services[i] = service
		}
		time.Sleep(4 * time.Second) // give some time to fw rules setup
		return ctx, nil
	})

	feature := features.New("External Service").
		Assess("the external DNS should be reachable via local service", func(ctx context.Context, t *testing.T) context.Context {
			for _, service := range services {
				for _, port := range service.Spec.Ports {
					ma.Logger.Info("Evaluating node port.", zap.Int32("nodeport", port.NodePort))
					reachability := manager.NewReachability(model.AllPods(), true)
					wrong := manager.ValidateOrFail(ma, model, &manager.TestCase{ToPort: int(port.NodePort), Protocol: v1.ProtocolTCP, Reachability: reachability})
					if wrong > 0 {
						t.Error("Wrong result number ")
					}
				}
			}
			return ctx
		}).Feature()

	externalEnv.AfterTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("after external service running")
		return ctx, nil
	})
	externalEnv.Test(ctx, t, feature)
}
