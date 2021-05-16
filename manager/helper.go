package manager

import (
	"go.uber.org/zap"
)

var (
	ignoreLoopback = false
)

// ValidateOrFail validates connectivity
func ValidateOrFail(k8s *KubeManager, model *Model, testCase *TestCase) int {
	var wrong int
	k8s.Logger.Info("Validating reachability matrix...")

	// 1st try
	k8s.Logger.Info("Validating reachability matrix... (== FIRST TRY ==)")
	ProbePodToPodConnectivity(k8s, model, testCase, false)

	// 2nd try, in case first one failed
	if _, wrong, _, _ = testCase.Reachability.Summary(ignoreLoopback); wrong != 0 {
		k8s.Logger.Info("failed first probe with wrong results ... retrying (== SECOND TRY ==)", zap.Int("wrong", wrong))
		ProbePodToPodConnectivity(k8s, model, testCase, false)
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
