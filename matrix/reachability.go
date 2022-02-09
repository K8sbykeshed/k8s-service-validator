package matrix

import (
	"fmt"

	"github.com/k8sbykeshed/k8s-service-validator/consts"
	"github.com/k8sbykeshed/k8s-service-validator/entities"
	"go.uber.org/zap"

	v1 "k8s.io/api/core/v1"
)

// TestCase describes the model for a netpol test
type TestCase struct {
	ToPort       int
	Protocol     v1.Protocol
	Reachability *Reachability
	ServiceType  string
}

// SetServiceType sets serviceType for the testCase
func (t *TestCase) SetServiceType(serviceType string) {
	t.ServiceType = serviceType
}

// GetServiceType returns ServiceType for the testCase
func (t *TestCase) GetServiceType() string {
	return t.ServiceType
}

// Reachability packages the model for a cluster-wide connectivity probe
type Reachability struct {
	Expected *TruthTable
	Observed *TruthTable
	Pods     []*entities.Pod
}

// NewReachability instantiates a reachability
func NewReachability(pods []*entities.Pod, defaultExpectation bool) *Reachability {
	podNames := make([]string, len(pods))
	for i, pod := range pods {
		podNames[i] = pod.PodString().String()
	}
	r := &Reachability{
		Expected: NewTruthTableFromItems(podNames, &defaultExpectation),
		Observed: NewTruthTableFromItems(podNames, nil),
		Pods:     pods,
	}
	return r
}

// PrintSummary prints the summary
func (r *Reachability) PrintSummary(printExpected, printObserved, printComparison, printBandwidth bool) {
	right, wrong, ignored, comparison := r.Summary(false, false)
	if ignored > 0 {
		zap.L().Warn(fmt.Sprintf("warning: this test doesn't take into consideration hairpin traffic, i.e. traffic whose source and destination is the same pod: %d cases ignored", ignored))
	}
	zap.L().Info(fmt.Sprintf("Reachability results (%t): correct: %v, incorrect: %v", wrong == 0, right, wrong))

	if printExpected {
		zap.L().Info(fmt.Sprintf("expected:\n\n%s\n\n\n", r.Expected.PrettyPrint("")))
	}
	if !printBandwidth && printObserved {
		zap.L().Info(fmt.Sprintf("observed:\n\n%s\n\n\n", r.Observed.PrettyPrint("")))
	}
	if printBandwidth {
		zap.L().Info(fmt.Sprintf("observed bandwidth:\n\n%s\n\n\n", r.Observed.PrettyPrintBandwidth("")))
	}
	if printComparison {
		zap.L().Info(fmt.Sprintf("comparison:\n\n%s\n\n\n", comparison.PrettyPrint("")))
	}
}

// Summary produces a useful summary of expected and observed model
func (r *Reachability) Summary(ignoreLoopback, measureBandWidth bool) (trueObs, falseObs, ignoredObs int, comparison *TruthTable) {
	falseObs, trueObs, ignoredObs = 0, 0, 0
	comparison = r.Expected.Compare(r.Observed)
	if !comparison.IsComplete() {
		fmt.Println("observations not complete!")
	}
	for from, dict := range comparison.Values {
		for to, val := range dict {
			if ignoreLoopback && from == to {
				// Never fail on loopback, because its not yet defined.
				ignoredObs++
			} else if val {
				if !measureBandWidth {
					trueObs++
				} else {
					connected := r.Observed.Values[from][to]
					bandwidth := r.Observed.Bandwidths[from][to]
					if connected && (bandwidth != nil && bandwidth.BandwidthToBytes() < consts.PerfTestBandWidthBenchMarkMegabytesPerSecond) {
						falseObs++
					} else {
						trueObs++
					}
				}
			} else {
				falseObs++
			}
		}
	}
	return
}

// Peer is used for matching pods by either or both of the pod's namespace and name.
type Peer struct {
	Namespace string
	Pod       string
}

// Matches checks whether the Peer matches the PodString:
// - an empty namespace means the namespace will always match
// - otherwise, the namespace must match the PodString's namespace
// - same goes for Pod: empty matches everything, otherwise must match exactly
func (p *Peer) Matches(pod entities.PodString) bool {
	return (p.Namespace == "" || p.Namespace == pod.Namespace()) && (p.Pod == "" || p.Pod == pod.PodName())
}

// ExpectPeer sets expected values using Peer matchers
func (r *Reachability) ExpectPeer(from, to *Peer, connected bool) {
	for _, fromPod := range r.Pods {
		if from.Matches(fromPod.PodString()) {
			for _, toPod := range r.Pods {
				if to.Matches(toPod.PodString()) {
					r.Expected.Set(string(fromPod.PodString()), string(toPod.PodString()), connected)
				}
			}
		}
	}
}

// Observe records a single connectivity observation
func (r *Reachability) Observe(fromPod, toPod entities.PodString, isConnected bool, bandwidth *ProbeJobBandwidthResults) {
	r.Observed.Set(string(fromPod), string(toPod), isConnected)
	r.Observed.SetBandwidth(string(fromPod), string(toPod), bandwidth)
}
