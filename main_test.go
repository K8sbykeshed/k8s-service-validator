package suites

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/K8sbykeshed/svc-tests/manager"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var (
	testenv env.Environment
	cs      *kubernetes.Clientset
	ctx     = context.Background()
)

// clientSet returns the Kubernetes clientset
func clientSet() *kubernetes.Clientset {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func TestMain(m *testing.M) {
	cs = clientSet()
	testenv = env.New()

	testenv.BeforeTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("====== before test")
		_, model, m := manager.GetModel(cs)
		if err := m.InitializeCluster(model); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	testenv.AfterTest(func(ctx context.Context) (context.Context, error) {
		fmt.Println("====== after test")
		time.Sleep(5 * time.Second)
		_, _, m := manager.GetModel(cs)
		if err := m.DeleteNamespaces([]string{"name-x"}); err != nil {
			log.Fatal(err)
		}
		return ctx, nil
	})

	testenv.Run(ctx, m)
}
