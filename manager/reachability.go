package manager

import (
	"fmt"
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/workload"
	v1 "k8s.io/api/core/v1"
)

// TestCase describes the data for a netpol test
type TestCase struct {
	ToPort       int
	Protocol     v1.Protocol
	Reachability *Reachability
}

// Reachability packages the data for a cluster-wide connectivity probe
type Reachability struct {
	Expected *TruthTable
	Observed *TruthTable
	Pods     []*workload.Pod
}

// NewReachability instantiates a reachability
func NewReachability(pods []*workload.Pod, defaultExpectation bool) *Reachability {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.PodString().String())
	}
	r := &Reachability{
		Expected: NewTruthTableFromItems(podNames, &defaultExpectation),
		Observed: NewTruthTableFromItems(podNames, nil),
		Pods:     pods,
	}
	return r
}

// PrintSummary prints the summary
func (r *Reachability) PrintSummary(printExpected bool, printObserved bool, printComparison bool) {
	right, wrong, ignored, comparison := r.Summary(ignoreLoopback)
	if ignored > 0 {
		fmt.Println(fmt.Printf("warning: this test doesn't take into consideration hairpin traffic, i.e. traffic whose source and destination is the same pod: %d cases ignored", ignored))
	}
	fmt.Println(fmt.Printf("reachability: correct:%v, incorrect:%v, result=%t\n\n", right, wrong, wrong == 0))
	if printExpected {
		fmt.Println(fmt.Printf("expected:\n\n%s\n\n\n", r.Expected.PrettyPrint("")))
	}
	if printObserved {
		fmt.Println(fmt.Printf("observed:\n\n%s\n\n\n", r.Observed.PrettyPrint("")))
	}
	if printComparison {
		fmt.Println(fmt.Printf("comparison:\n\n%s\n\n\n", comparison.PrettyPrint("")))
	}
}

// Summary produces a useful summary of expected and observed data
func (r *Reachability) Summary(ignoreLoopback bool) (trueObs int, falseObs int, ignoredObs int, comparison *TruthTable) {
	comparison = r.Expected.Compare(r.Observed)
	if !comparison.IsComplete() {
		fmt.Println("observations not complete!")
	}
	falseObs, trueObs, ignoredObs = 0, 0, 0
	for from, dict := range comparison.Values {
		for to, val := range dict {
			if ignoreLoopback && from == to {
				// Never fail on loopback, because its not yet defined.
				ignoredObs++
			} else if val {
				trueObs++
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
func (p *Peer) Matches(pod workload.PodString) bool {
	return (p.Namespace == "" || p.Namespace == pod.Namespace()) && (p.Pod == "" || p.Pod == pod.PodName())
}

// ExpectPeer sets expected values using Peer matchers
func (r *Reachability) ExpectPeer(from *Peer, to *Peer, connected bool) {
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
func (r *Reachability) Observe(fromPod workload.PodString, toPod workload.PodString, isConnected bool) {
	r.Observed.Set(string(fromPod), string(toPod), isConnected)
}
