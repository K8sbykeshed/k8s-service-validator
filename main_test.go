package suites

import (
	"context"
	"fmt"
	"github.com/K8sbykeshed/svc-tests/manager"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"testing"
)

var (
	cs      *kubernetes.Clientset
	config  *rest.Config
	testenv env.Environment
	model   *manager.Model
	ma      *manager.KubeManager
	namespace string
	ctx     = context.Background()
)

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

func TestMain(m *testing.M) {
	cs, config := clientSet()
	testenv = env.New()
	testenv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		namespace, model, ma = manager.GetModel(cs, config)
		fmt.Println("====== before test")
		if err := ma.InitializeCluster(model); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	testenv.AfterTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("====== after test")
		//if err := ma.DeleteNamespaces([]string{namespace}); err != nil {
		//	log.Fatal(err)
		//}
		return ctx, nil
	})

	testenv.Run(ctx, m)
}
