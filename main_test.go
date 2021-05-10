package suites

import (
	"context"
	"fmt"
	"github.com/k8sbykeshed/svc-tests/manager"
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
	logger, _ = zap.NewProduction()
}

// getNamespaces returns a random namespace starting on x
func getNamespaces() (string, []string) {
	rand.Seed(time.Now().UnixNano())
	nsX := fmt.Sprintf("x-%d", rand.Intn(1e5))
	return nsX, []string{nsX}
}

func StartModel() (string, *manager.Model) {
	domain := "cluster.local"
	nsX, namespaces := getNamespaces()
	model := manager.NewModel(namespaces, []string{"a", "b", "c"}, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP}, domain)
	return nsX, model
}

func TestMain(m *testing.M) {
	testenv = env.New()
	cs, config = clientSet()
	ma = manager.NewKubeManager(cs, config, logger)

	testenv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		namespace, model = StartModel()
		if err := ma.InitializeCluster(model); err != nil {
			log.Fatal(err)
		}
		ma.WaitForHTTPServers(model)
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
