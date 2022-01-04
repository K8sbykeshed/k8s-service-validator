package matrix

import (
	"fmt"
	"k8s.io/client-go/util/homedir"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetNamespace returns a random namespace starting on x
func GetNamespace() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("x-%d", rand.Intn(1e5))
}

// NewClientSet returns the Kubernetes clientset
func NewClientSet() (*kubernetes.Clientset, *rest.Config) {
	var config *rest.Config
	kubeconfig, exists := os.LookupEnv("KUBECONFIG")
	if !exists {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	} else {

	}
	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, config
}

// ValidateOrFail validates connectivity
func ValidateOrFail(k8s *KubeManager, model *Model, testCase *TestCase, ignoreLoopback, reachTargetPod bool) int {
	var wrong int

	// 1st try
	k8s.Logger.Info("Validating reachability matrix... (== FIRST TRY ==)")
	ProbePodToPodConnectivity(k8s, model, testCase, reachTargetPod)

	// 2nd try, in case first one failed
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		k8s.Logger.Warn("Retrying (== SECOND TRY ==) - failed first probe with wrong results ... ", zap.Int("wrong", wrong))
		ProbePodToPodConnectivity(k8s, model, testCase, reachTargetPod)
	}

	// at this point we know if we passed or failed, print final matrix and pass/fail the test.
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		testCase.Reachability.PrintSummary(true, true, true)
		k8s.Logger.Info("Had %d wrong results in reachability matrix", zap.Int("wrong", wrong))
	}
	testCase.Reachability.PrintSummary(true, true, true)

	if wrong == 0 {
		k8s.Logger.Info("Tests passed, validation succeeded!")
	}
	return wrong
}

// todo(knabben) - make a generic in slice contains function
func protocolOnSlice(value v1.Protocol, slice []v1.Protocol) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func intOnSlice(value int32, slice []int32) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
