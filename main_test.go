package suites

import (
	"context"
	"fmt"
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"path/filepath"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"testing"
	"time"
)

var (
	namespace string
	config    *rest.Config
	testenv   env.Environment
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

// getNamespaces returns a random namespace starting on x
func getNamespaces() (string, []string) {
	rand.Seed(time.Now().UnixNano())
	nsX := fmt.Sprintf("x-%d", rand.Intn(1e5))
	return nsX, []string{nsX}
}

func StartModel(nodesLen int) (string, *manager.Model) {
	domain := "cluster.local"
	nsX, namespaces := getNamespaces()

	// Generate pod names using existing nodes
	var pods []string
	for i := 1; i <= nodesLen; i++ {
		pods = append(pods, fmt.Sprintf("pod-%d", i))
	}

	model := manager.NewModel(namespaces, pods, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP, v1.ProtocolUDP}, domain)
	return nsX, model
}

func TestMain(m *testing.M) {
	testenv = env.New()
	cs, config = clientSet()

	ma = manager.NewKubeManager(cs, config, logger)
	nodes, err := ma.GetReadyNodes()
	if err != nil {
		log.Fatal(err)
	}

	testenv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		namespace, model = StartModel(len(nodes))
		if err := ma.InitializeCluster(model, nodes); err != nil {
			log.Fatal(err)
		}
		if err := ma.WaitForHTTPServers(model); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	testenv.AfterTest(func(ctx context.Context) (context.Context, error) {
		if err := ma.DeleteNamespaces([]string{namespace}); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	testenv.Run(ctx, m)
}

// clientSet returns the Kubernetes clientset
func clientSet() (*kubernetes.Clientset, *rest.Config) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, config
}
