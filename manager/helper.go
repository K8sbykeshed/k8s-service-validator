package manager

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var ignoreLoopback = false

// GetNamespace returns a random namespace starting on x
func GetNamespace() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("x-%d", rand.Intn(1e5))
}

// NewClientSet returns the Kubernetes clientset
func NewClientSet() (*kubernetes.Clientset, *rest.Config) {
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

// ValidateOrFail validates connectivity
func ValidateOrFail(k8s *KubeManager, model *Model, testCase *TestCase) int {
	var wrong int
	k8s.Logger.Info("Validating reachability matrix...")

	// 1st try
	k8s.Logger.Info("Validating reachability matrix... (== FIRST TRY ==)")
	ProbePodToPodConnectivity(k8s, model, testCase)

	// 2nd try, in case first one failed
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		k8s.Logger.Info("Retrying (== SECOND TRY ==) - failed first probe with wrong results ... ", zap.Int("wrong", wrong))
		ProbePodToPodConnectivity(k8s, model, testCase)
	}

	// at this point we know if we passed or failed, print final matrix and pass/fail the test.
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		testCase.Reachability.PrintSummary(true, true, true)
		k8s.Logger.Info("Had %d wrong results in reachability matrix", zap.Int("wrong", wrong))
	}
	testCase.Reachability.PrintSummary(true, true, true)

	if wrong == 0 {
		k8s.Logger.Info("VALIDATION SUCCESSFUL")
	}
	return wrong
}
