package manager

import (
	"fmt"
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
func probeWorker(manager *KubeManager, jobs <-chan *ProbeJob, results chan<- *ProbeJobResults) {
	for job := range jobs {
		podFrom := job.PodFrom

		connected, command, err := manager.probeConnectivity(
			podFrom.Namespace, podFrom.Name, podFrom.Containers[0].Name(), job.PodTo.QualifiedServiceAddress(job.ToPodDNSDomain), job.Protocol, job.ToPort,
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
func ProbePodToPodConnectivity(k8s *KubeManager, model *Model, testCase *TestCase) {
	numberOfWorkers := 3 // See https://github.com/kubernetes/kubernetes/pull/97690
	allPods := model.AllPods()
	size := len(allPods) * len(allPods)
	jobs := make(chan *ProbeJob, size)
	results := make(chan *ProbeJobResults, size)
	for i := 0; i < numberOfWorkers; i++ {
		go probeWorker(k8s, jobs, results)
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
			fmt.Println(fmt.Printf("unable to perform probe %s -> %s: %v", job.PodFrom.PodString(), job.PodTo.PodString(), result.Err))
		}
		testCase.Reachability.Observe(job.PodFrom.PodString(), job.PodTo.PodString(), result.IsConnected)
		expected := testCase.Reachability.Expected.Get(job.PodFrom.PodString().String(), job.PodTo.PodString().String())

		if result.IsConnected != expected {
			fmt.Println(fmt.Printf("Validation of %s -> %s FAILED !!!", job.PodFrom.PodString(), job.PodTo.PodString()))
			fmt.Println(fmt.Printf("error %v ", result.Err))

			if expected {
				fmt.Println(fmt.Printf("Expected allowed pod connection was instead BLOCKED --- run '%v'", result.Command))
			} else {
				fmt.Println(fmt.Printf("Expected blocked pod connection was instead ALLOWED --- run '%v'", result.Command))
			}
		}
	}
}