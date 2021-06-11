package manager

import (
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/data"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// ProbeJob packages the data for the input of a pod->pod connectivity probe
type ProbeJob struct {
	PodFrom        *data.Pod
	PodTo          *data.Pod
	ToPort         int
	ToPodDNSDomain string
	Protocol       v1.Protocol
	ServiceType    string
}

func (p *ProbeJob) SetServiceType(serviceType string) {
	p.ServiceType = serviceType
}

func (p *ProbeJob) GetServiceType() string {
	return p.ServiceType
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
		var addrTo string

		// Choose the host and port based on service or probing
		switch job.GetServiceType() {
		case data.PodIP:
			addrTo = job.PodTo.GetPodIP() // use pod IP for probing proposes only
		case data.ClusterIP:
			addrTo = job.PodTo.QualifiedServiceAddress(job.ToPodDNSDomain)
		case data.NodePort:
			addrTo = job.PodTo.GetHostIP()
		case data.LoadBalancer:
			addrTo = job.PodTo.GetExternalIP()
		default:
			addrTo = job.PodTo.GetPodIP()
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
			// if testcase global toPort not set, fallbacks to Pod custom set Port.
			toPort := testCase.ToPort
			if toPort == 0 {
				toPort = int(podTo.GetToPort())
			}

			jobs <- &ProbeJob{
				PodFrom:        podFrom,
				PodTo:          podTo,
				ToPort:         toPort,
				ToPodDNSDomain: model.DNSDomain,
				Protocol:       testCase.Protocol,
				ServiceType:    testCase.ServiceType,
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

		k8s.Logger.Info(result.Command)
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
