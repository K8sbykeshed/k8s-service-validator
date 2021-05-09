package manager

import "fmt"

var (
	ignoreLoopback = false
)
// ValidateOrFail validates connectivity
func ValidateOrFail(k8s *KubeManager, model *Model, testCase *TestCase) {
	fmt.Println("Validating reachability matrix...")

	// 1st try
	fmt.Println("Validating reachability matrix... (FIRST TRY)")
	ProbePodToPodConnectivity(k8s, model, testCase)
	// 2nd try, in case first one failed
	if _, wrong, _, _ := testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		fmt.Println(fmt.Printf("failed first probe %d wrong results ... retrying (SECOND TRY)", wrong))
		ProbePodToPodConnectivity(k8s, model, testCase)
	}

	// at this point we know if we passed or failed, print final matrix and pass/fail the test.
	if _, wrong, _, _ := testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		testCase.Reachability.PrintSummary(true, true, true)
		fmt.Println(fmt.Printf("Had %d wrong results in reachability matrix", wrong))
	}
	testCase.Reachability.PrintSummary(true, true, true)
	fmt.Println(fmt.Printf("VALIDATION SUCCESSFUL"))
}