package manager

import (
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)


// ProbeJob packages the data for the input of a pod->pod connectivity probe
type ProbeJob struct {
	PodFrom        *Pod
	PodTo          *Pod
	ToPort         int
	ToPodDNSDomain string
	Protocol       v1.Protocol
}

// ProbeJobResults packages the data for the results of a pod->pod connectivity probe
type ProbeJobResults struct {
	Job         *ProbeJob
	IsConnected bool
	Err         error
	Command     string
}


// probeWorker continues polling a pod connectivity status, until the incoming "jobs" channel is closed, and writes results back out to the "results" channel.
// it only writes pass/fail status to a channel and has no failure side effects, this is by design since we do not want to fail inside a goroutine.
func probeWorker(manager *KubeManager, jobs <-chan *ProbeJob, results chan<- *ProbeJobResults, usePodIP bool) {
	for job := range jobs {

		addrTo := job.PodTo.QualifiedServiceAddress(job.ToPodDNSDomain)
		if usePodIP {  // used for initial probing (no services exists yet)
			addrTo = job.PodTo.PodIP
		} else if job.ToPort > 30000 {  // node port
			addrTo = job.PodTo.HostIP
		}

		podFrom := job.PodFrom
		connected, command, err := manager.probeConnectivity(
			podFrom.Namespace, podFrom.Name, podFrom.Containers[0].Name(), addrTo, job.Protocol, job.ToPort,
		)

		result := &ProbeJobResults{
			Job:         job,
			IsConnected: connected,
			Err:         err,
			Command:     command,
		}
		results <- result
	}
}

// ProbePodToPodConnectivity runs a series of probes in kube, and records the results in `testCase.Reachability`
func ProbePodToPodConnectivity(k8s *KubeManager, model *Model, testCase *TestCase, usePodIP bool) {
	numberOfWorkers := 3 // See https://github.com/kubernetes/kubernetes/pull/97690
	allPods := model.AllPods()
	size := len(allPods) * len(allPods)
	jobs := make(chan *ProbeJob, size)
	results := make(chan *ProbeJobResults, size)
	for i := 0; i < numberOfWorkers; i++ {
		go probeWorker(k8s, jobs, results, usePodIP)
	}

	for _, podFrom := range allPods {
		for _, podTo := range allPods {
			jobs <- &ProbeJob{
				PodFrom:        podFrom,
				PodTo:          podTo,
				ToPort:         testCase.ToPort,
				ToPodDNSDomain: model.DNSDomain,
				Protocol:       testCase.Protocol,
			}
		}
	}
	close(jobs)

	for i := 0; i < size; i++ {
		result := <-results
		job := result.Job
		if result.Err != nil {
			k8s.Logger.Warn("Unable to perform probe.",
				zap.String("from", string(job.PodFrom.PodString())),
				zap.String("to", string(job.PodTo.PodString())),
			)
			if result.Err != nil {
				k8s.Logger.Warn("ERROR", zap.String("err", result.Err.Error()))
			}
		}
		testCase.Reachability.Observe(job.PodFrom.PodString(), job.PodTo.PodString(), result.IsConnected)
		expected := testCase.Reachability.Expected.Get(job.PodFrom.PodString().String(), job.PodTo.PodString().String())

		if result.IsConnected != expected {
			k8s.Logger.Warn("Validation FAILED!",
				zap.String("from", string(job.PodFrom.PodString())),
				zap.String("to", string(job.PodTo.PodString())),
			)
			if result.Err != nil {
				k8s.Logger.Warn("ERROR", zap.String("err", result.Err.Error()))
			}
			if expected {
				k8s.Logger.Warn("Expected allowed pod connection was instead BLOCKED", zap.String("result", result.Command))
			} else {
				k8s.Logger.Warn("Expected blocked pod connection was instead ALLOWED", zap.String("result", result.Command))
			}
		}
	}
}