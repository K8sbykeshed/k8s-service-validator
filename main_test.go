package suites

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/k8sbykeshed/k8s-service-lb-validator/manager"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/e2e-framework/pkg/env"
)

var (
	config    *rest.Config
	testenv   env.Environment
	namespace string
	model     *manager.Model
	cs        *kubernetes.Clientset
	ma        *manager.KubeManager
	ctx       = context.Background()
	logger    *zap.Logger
)

func init() {
	var err error
	if logger, err = zap.NewProduction(); err != nil {
		log.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var (
		err   error
		nodes []*v1.Node
	)

	testenv = env.New()
	cs, config = manager.NewClientSet()
	namespace = manager.GetNamespace()
	namespaces := []string{namespace}

	// Create a new Manager to control K8S resources.
	ma = manager.NewKubeManager(cs, config, logger)
	if nodes, err = ma.GetReadyNodes(); err != nil {
		log.Fatal(err)
	}

	// Setup brings the pods only in the start, all tests share the same pods
	// access them via different services types.
	testenv.Setup(func(ctx context.Context) (context.Context, error) {
		// Generate pod names using existing nodes
		var pods []string
		for i := 1; i <= len(nodes); i++ {
			pods = append(pods, fmt.Sprintf("pod-%d", i))
		}

		// Initialize the model and cluster.
		domain := "cluster.local"
		model = manager.NewModel(namespaces, pods, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP}, domain)
		if err = ma.InitializeCluster(model, nodes); err != nil {
			log.Fatal(err)
		}

		// Wait for servers to be up.
		if err = ma.WaitForHTTPServers(model); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	// Finished cleans up the namespace in the end of the suite.
	testenv.Finish(func(ctx context.Context) (context.Context, error) {
		logger.Info("Cleanup namespace.", zap.String("namespace", namespace))
		if err := ma.DeleteNamespaces(namespaces); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})
	testenv.Run(ctx, m)
}
