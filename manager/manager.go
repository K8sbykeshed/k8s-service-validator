package manager

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/data"
)

const (
	waitInterval = 1 * time.Second
)

// KubeManager is the core struct to manage kubernetes objects
type KubeManager struct {
	config    *rest.Config
	Logger    *zap.Logger
	clientSet *kubernetes.Clientset
}

// NewKubeManager returns a new KubeManager
func NewKubeManager(cs *kubernetes.Clientset, config *rest.Config, logger *zap.Logger) *KubeManager {
	return &KubeManager{clientSet: cs, config: config, Logger: logger}
}

// InitializeCluster start all pods and wait them to be up
func (k *KubeManager) InitializeCluster(model *Model, nodes []*v1.Node) error {
	k.Logger.Info("Initializing Pods in the cluster.")

	var createdPods []*v1.Pod
	for _, ns := range model.Namespaces { // create namespaces
		if _, err := k.CreateNamespace(ns.Spec()); err != nil {
			return err
		}

		// Check size of nodes and already modeled pods
		if len(ns.Pods) != len(nodes) || len(nodes) <= 1 {
			return errors.Errorf("invalid number of %d nodes.", len(nodes))
		}

		for i, pod := range ns.Pods {
			// Set NodeName on pods being created
			pod.NodeName = nodes[i].Name
			k.Logger.Info("creating/updating pod.",
				zap.String("namespace", ns.Name),
				zap.String("name", pod.Name),
				zap.String("node", pod.NodeName),
			)
			kubePod, err := k.createPod(pod.KubePod())
			if err != nil {
				return err
			}
			createdPods = append(createdPods, kubePod)
		}
	}

	for _, podString := range model.AllPodStrings() {
		kubePod, err := k.getPod(podString.Namespace(), podString.PodName())
		if err != nil {
			return err
		}
		if kubePod == nil {
			return errors.Errorf("unable to find pod in ns %s with key/val pod=%s", podString.Namespace(), podString.PodName())
		}

		k.Logger.Info("Wait for pod running.", zap.String("name", kubePod.Name), zap.String("namespace", kubePod.Namespace))
		err = data.WaitForPodNameRunningInNamespace(k.clientSet, kubePod.Name, kubePod.Namespace)
		if err != nil {
			return errors.Wrapf(err, "unable to wait for pod %s/%s", podString.Namespace(), podString.PodName())
		}
	}

	pods := model.AllPods()
	for i, createdPod := range createdPods {
		kubePod, err := k.getPod(createdPod.Namespace, createdPod.Name)
		if err != nil {
			return err
		}
		k.Logger.Info("Wait for pod running.", zap.String("name", kubePod.Name), zap.String("namespace", kubePod.Namespace))
		if err := data.WaitForPodRunningInNamespace(k.clientSet, createdPod); err != nil {
			return errors.Wrapf(err, "unable to wait for pod %s/%s", createdPod.Namespace, createdPod.Name)
		}
		// Set IP addresses on Pod model.
		pods[i].SetPodIP(kubePod.Status.PodIP)
		pods[i].SetHostIP(kubePod.Status.HostIP)
	}
	return nil
}

// createPod is a convenience function for pod setup.
func (k *KubeManager) createPod(pod *v1.Pod) (*v1.Pod, error) {
	ns := pod.Namespace
	createdPod, err := k.clientSet.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update pod %s/%s", ns, pod.Name)
	}
	return createdPod, nil
}


func (k *KubeManager) CreateNamespace(ns *v1.Namespace) (*v1.Namespace, error) {
	createdNamespace, err := k.clientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update namespace %s", ns.Name)
	}
	return createdNamespace, nil
}

// DeleteNamespaces
func (k *KubeManager) DeleteNamespaces(namespaces []string) error {
	for _, ns := range namespaces {
		err := k.clientSet.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to delete namespace %s", ns)
		}
	}
	return nil
}

// DeleteServices
func (k *KubeManager) DeleteServices(services []*v1.Service) error {
	for _, svc := range services {
		name := svc.Name
		err := k.clientSet.CoreV1().Services(svc.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to delete service %s", name)
		}
	}
	return nil
}

// GetReadyNodes returns the ready nodes in the cluster
func (k *KubeManager) GetReadyNodes() ([]*v1.Node, error) {
	kubeNode, err := k.clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}

	// filter in the ready nodes.
	var nodes []*v1.Node
	for _, node := range kubeNode.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady {
				if cond.Status == v1.ConditionTrue {
					nodes = append(nodes, node.DeepCopy())
				}
			}
		}
	}
	return nodes, nil
}

// GetLoadBalancerServices returns the external-ips from load-balancer servicess
func (k *KubeManager) GetLoadBalancerService(svc *v1.Service) ([]string, error) {
	ips := []string{}
	kubeService, err := k.clientSet.CoreV1().Services(svc.Namespace).Get(context.TODO(), svc.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get service %s/%s", svc.Namespace, svc.Name)
	}
	for _, ip := range kubeService.Status.LoadBalancer.Ingress {
		ips = append(ips, ip.IP)
	}
	return ips, nil
}

// getPod gets a pod by namespace and name.
func (k *KubeManager) getPod(ns, name string) (*v1.Pod, error) {
	kubePod, err := k.clientSet.CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get pod %s/%s", ns, name)
	}
	return kubePod, nil
}

// probeConnectivity execs into a pod and checks its connectivity to another pod..
func (k *KubeManager) probeConnectivity(nsFrom, podFrom, containerFrom, addrTo string, protocol v1.Protocol, toPort int) (bool, string, error) {
	port := strconv.Itoa(toPort)
	var cmd []string
	switch protocol {
	case v1.ProtocolSCTP:
		cmd = []string{"/agnhost", "connect", net.JoinHostPort(addrTo, port), "--timeout=5s", "--protocol=sctp"}
	case v1.ProtocolTCP:
		cmd = []string{"/agnhost", "connect", net.JoinHostPort(addrTo, port), "--timeout=5s", "--protocol=tcp"}
	case v1.ProtocolUDP:
		cmd = []string{"/agnhost", "connect", net.JoinHostPort(addrTo, port), "--timeout=5s", "--protocol=udp"}
	default:
		fmt.Println(fmt.Printf("protocol %s not supported", protocol))
	}

	commandDebugString := fmt.Sprintf("kubectl exec %s -c %s -n %s -- %s", podFrom, containerFrom, nsFrom, strings.Join(cmd, " "))

	stdout, stderr, err := k.executeRemoteCommand(nsFrom, podFrom, containerFrom, cmd)
	if err != nil {
		fmt.Println(fmt.Printf("%s/%s -> %s: error when running command: err - %v /// stdout - %s /// stderr - %s", nsFrom, podFrom, addrTo, err, stdout, stderr))
		return false, commandDebugString, nil
	}
	return true, commandDebugString, nil
}

// executeRemoteCommand executes a remote shell command on the given pod.
func (k *KubeManager) executeRemoteCommand(namespace, pod, containerName string, command []string) (string, string, error) {
	return data.ExecWithOptions(k.config, k.clientSet, &data.ExecOptions{
		Command:            command,
		Namespace:          namespace,
		PodName:            pod,
		ContainerName:      containerName,
		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}

// WaitForHTTPServers waits for all webservers to be up, on all protocols, and then validates them using the same probe logic as the rest of the suite.
func (k *KubeManager) WaitForHTTPServers(model *Model) error {
	const maxTries = 10
	k.Logger.Info("Waiting for HTTP servers (ports 80 and 81) to become ready")

	testCases := map[string]*TestCase{}
	for _, port := range model.Ports {
		for _, protocol := range model.Protocols {
			fromPort := 81
			desc := fmt.Sprintf("%d->%d,%s", fromPort, port, protocol)
			testCases[desc] = &TestCase{ToPort: int(port), Protocol: protocol, ServiceType: data.PodIP}
		}
	}
	notReady := map[string]bool{}
	for caseName := range testCases {
		notReady[caseName] = true
	}

	for i := 0; i < maxTries; i++ {
		for caseName, testCase := range testCases {
			if notReady[caseName] {
				reachability := NewReachability(model.AllPods(), true)
				testCase.Reachability = reachability
				ProbePodToPodConnectivity(k, model, testCase)
				_, wrong, _, _ := reachability.Summary(false)
				if wrong == 0 {
					k.Logger.Info("Server is ready", zap.String("case", caseName))
					delete(notReady, caseName)
				} else {
					k.Logger.Info("Server is not ready", zap.String("case", caseName))
				}
			}
		}
		if len(notReady) == 0 {
			return nil
		}
		time.Sleep(waitInterval)
	}
	return errors.Errorf("after %d tries, %d HTTP servers are not ready", maxTries, len(notReady))
}
