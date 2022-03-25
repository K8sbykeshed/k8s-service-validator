package matrix

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/k8sbykeshed/k8s-service-validator/pkg/entities"
)

// GetNamespace returns a random namespace starting on x
func GetNamespace() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("x-%d", rand.Intn(1e5))
}

// GetIPerfNamespace returns a random namespace starting on x and ends with IPerfNamespaceSuffix
func GetIPerfNamespace() string {
	ns := GetNamespace()
	return ns + entities.IPerfNamespaceSuffix
}

// NewClientSet returns the Kubernetes clientset
func NewClientSet() (*kubernetes.Clientset, *rest.Config) {
	var config *rest.Config
	kubeconfig, exists := os.LookupEnv("KUBECONFIG")
	if !exists {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
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
	return ValidateAndMeasureBandwidthOrFail(k8s, model, testCase, ignoreLoopback, reachTargetPod, false)
}

// ValidateAndMeasureBandwidthOrFail validates connectivity and also measure bandwidth
// if measureBandWidth is true
func ValidateAndMeasureBandwidthOrFail(k8s *KubeManager, model *Model, testCase *TestCase, ignoreLoopback,
	reachTargetPod, measureBandWidth bool,
) int {
	var wrong int

	// 1st try
	zap.L().Info("Validating reachability matrix, first try.")
	ProbePodToPodConnectivity(k8s, model, testCase, reachTargetPod, measureBandWidth)

	// 2nd try, in case first one failed
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback, measureBandWidth); wrong != 0 {
		zap.L().Warn("Failed first probe with wrong results, retrying...", zap.Int("wrong", wrong))
		ProbePodToPodConnectivity(k8s, model, testCase, reachTargetPod, measureBandWidth)
	}

	// at this point we know if we passed or failed, print final matrix and pass/fail the test.
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback, measureBandWidth); wrong != 0 {
		testCase.Reachability.PrintSummary(true, true, true, measureBandWidth)
		zap.L().Info("Had wrong results in reachability matrix", zap.Int("wrong", wrong))
	}
	testCase.Reachability.PrintSummary(true, true, true, measureBandWidth)

	if wrong == 0 {
		zap.L().Info("Tests passed, validation succeeded!")
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
