package matrix

import (
	"github.com/k8sbykeshed/k8s-service-validator/entities"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// ProbeJob packages the model for the input of a pod->pod connectivity probe
type ProbeJob struct {
	PodFrom        *entities.Pod
	PodTo          *entities.Pod
	ToPort         int
	ToPodDNSDomain string
	Protocol       v1.Protocol
	ServiceType    string
	// ReachTargetPod checks if connection is responded by the target pod, instead of only check successful connection
	ReachTargetPod bool
}

// SetServiceType sets the ServiceType for the probeJob
func (p *ProbeJob) SetServiceType(serviceType string) {
	p.ServiceType = serviceType
}

// GetServiceType returns ServiceType for the probeJob
func (p *ProbeJob) GetServiceType() string {
	return p.ServiceType
}

// ProbeJobResults packages the model for the results of a pod->pod connectivity probe
type ProbeJobResults struct {
	Job         *ProbeJob
	IsConnected bool
	Err         error
	Command     string
	Endpoint    string
}

// probeWorker continues polling a pod connectivity status, until the incoming "jobs" channel is closed, and writes results back out to the "results" channel.
// it only writes pass/fail status to a channel and has no failure side effects, this is by design since we do not want to fail inside a goroutine.
func probeWorker(manager *KubeManager, jobs <-chan *ProbeJob, results chan<- *ProbeJobResults) {
	for job := range jobs {
		var addrTo string

		if job.PodTo.SkipProbe {
			results <- &ProbeJobResults{
				Job:         job,
				IsConnected: true,
				Err:         nil,
				Command:     "skip",
			}
			return
		}

		// Choose the host and port based on service or probing
		switch job.GetServiceType() {
		case entities.PodIP:
			addrTo = job.PodTo.GetPodIP()
		case entities.ClusterIP:
			addrTo = job.PodTo.GetClusterIP()
		case entities.NodePort:
			addrTo = job.PodTo.GetHostIP()
		case entities.ExternalName:
			addrTo = job.PodTo.GetServiceName()
		case entities.LoadBalancer:
			var externalIPs []entities.ExternalIP
			if job.Protocol == v1.ProtocolTCP {
				externalIPs = job.PodTo.GetExternalIPsByProtocol(v1.ProtocolTCP)
			} else if job.Protocol == v1.ProtocolUDP {
				externalIPs = job.PodTo.GetExternalIPsByProtocol(v1.ProtocolUDP)
			}
			// Temporary solution to unblock the tests, load balancer IPs take longer time than expected to get created.
			// will solve in https://github.com/K8sbykeshed/k8s-service-validator/issues/44
			if len(externalIPs) > 0 {
				addrTo = externalIPs[0].IP
			}

		default:
			addrTo = job.PodTo.GetPodIP()
		}

		podFrom := job.PodFrom
		var connected bool
		var command string
		var err error
		var ep string
		if job.ReachTargetPod {
			connected, ep, command, err = manager.ProbeConnectivityWithCurl(
				podFrom.Namespace, podFrom.Name, podFrom.Containers[0].GetName(), addrTo, job.Protocol, job.ToPort,
			)
		} else {
			connected, command, err = manager.ProbeConnectivity(
				podFrom.Namespace, podFrom.Name, podFrom.Containers[0].GetName(), addrTo, job.Protocol, job.ToPort,
			)
		}
		if job.ReachTargetPod && job.PodTo.Name != ep {
			connected = false
		}
		result := &ProbeJobResults{
			Job:         job,
			IsConnected: connected,
			Err:         err,
			Command:     command,
			Endpoint:    ep,
		}
		results <- result
	}
}

// ProbePodToPodConnectivity runs a series of probes in kube, and records the results in `testCase.Reachability`
func ProbePodToPodConnectivity(k8s *KubeManager, model *Model, testCase *TestCase, reachTargetPod bool) {
	numberOfWorkers := 4 // See https://github.com/kubernetes/kubernetes/pull/97690
	allPods := model.AllPods()
	size := len(allPods) * len(allPods)
	jobs := make(chan *ProbeJob, size)
	results := make(chan *ProbeJobResults, size)
	for i := 1; i < numberOfWorkers; i++ {
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
				ToPodDNSDomain: model.dnsDomain,
				Protocol:       testCase.Protocol,
				ServiceType:    testCase.ServiceType,
				ReachTargetPod: reachTargetPod,
			}
		}
	}
	close(jobs)

	for i := 0; i < size; i++ {
		result := <-results
		job := result.Job
		if result.Err != nil {
			k8s.Logger.Error("Unable to perform probe.",
				zap.String("from", string(job.PodFrom.PodString())),
				zap.String("to", string(job.PodTo.PodString())),
			)
			if result.Err != nil {
				k8s.Logger.Warn("ERROR", zap.String("err", result.Err.Error()))
			}
		}

		fields := []zap.Field{
			zap.String("from", string(job.PodFrom.PodString())),
			zap.String("to", string(job.PodTo.PodString())),
			zap.String("cmd", result.Command),
		}
		if job.PodTo.SkipProbe {
			k8s.Logger.Debug("Skipping probe", fields...)
		} else {
			k8s.Logger.Debug("Validating matrix.", fields...)
		}

		testCase.Reachability.Observe(job.PodFrom.PodString(), job.PodTo.PodString(), result.IsConnected)
		expected := testCase.Reachability.Expected.Get(job.PodFrom.PodString().String(), job.PodTo.PodString().String())

		if result.IsConnected != expected {
			k8s.Logger.Error("Connection blocked!",
				zap.String("from", string(job.PodFrom.PodString())),
				zap.String("to", string(job.PodTo.PodString())),
			)
			if result.Err != nil {
				k8s.Logger.Error("ERROR", zap.String("err", result.Err.Error()))
			}
			if expected {
				k8s.Logger.Warn("Expected allowed pod connection was instead BLOCKED", zap.String("result", result.Command))
			} else {
				k8s.Logger.Warn("Expected blocked pod connection was instead ALLOWED", zap.String("result", result.Command))
			}
		}
	}
}
