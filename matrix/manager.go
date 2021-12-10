package matrix

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
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

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	k8s "github.com/k8sbykeshed/k8s-service-validator/entities/kubernetes"
)

const (
	waitInterval = 1 * time.Second
)

// KubeManager is the core struct to manage kubernetes entities
type KubeManager struct {
	config    *rest.Config
	Logger    *zap.Logger
	clientSet *kubernetes.Clientset
}

// NewKubeManager returns a new KubeManager
func NewKubeManager(cs *kubernetes.Clientset, config *rest.Config, logger *zap.Logger) *KubeManager {
	return &KubeManager{clientSet: cs, config: config, Logger: logger}
}

// StartPods start all pods and wait them to be up
func (k *KubeManager) StartPods(model *Model, nodes []*v1.Node) error {
	k.Logger.Info("Initializing Pods in the cluster.")
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
			k.Logger.Debug("creating/updating pod.",
				zap.String("namespace", ns.Name),
				zap.String("name", pod.Name),
				zap.String("node", pod.NodeName),
			)
			if _, err := k.CreatePod(pod.ToK8SSpec()); err != nil {
				return err
			}
		}
	}
	// waiting for pods running.
	for _, createdPod := range model.AllPods() {
		if err := k.WaitAndSetIPs(createdPod); err != nil {
			return err
		}
	}
	return nil
}

// WaitAndSetIPs wait for running pods and set internal pod and host IP addresses.
func (k *KubeManager) WaitAndSetIPs(modelPod *entities.Pod) error {
	var err error

	kubePod := modelPod.ToK8SSpec()
	k.Logger.Debug("Wait for pod running.", zap.String("name", modelPod.Name), zap.String("namespace", modelPod.Namespace))

	if err := k8s.WaitForPodRunningInNamespace(k.clientSet, kubePod); err != nil {
		return errors.Wrapf(err, "unable to wait for pod %s/%s", modelPod.Namespace, modelPod.Name)
	}
	if kubePod, err = k.GetPod(modelPod.Namespace, modelPod.Name); err != nil {
		return err
	}

	// Set IP addresses on Pod model.
	modelPod.SetPodIP(kubePod.Status.PodIP)
	modelPod.SetHostIP(kubePod.Status.HostIP)
	return nil
}

// CreatePod is a convenience function for pod setup
func (k *KubeManager) CreatePod(podSpec *v1.Pod) (*v1.Pod, error) {
	nsName := podSpec.Namespace
	pod, err := k.clientSet.CoreV1().Pods(nsName).Create(context.TODO(), podSpec, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create pod %s/%s", nsName, podSpec.Name)
	}
	return pod, nil
}

// AddLabelToPod adds a label to a pod
func (k *KubeManager) AddLabelToPod(podSpec *entities.Pod, key string, value string)  error {
	nsName := podSpec.Namespace

	_, err := k.clientSet.CoreV1().Pods(nsName).Patch(context.TODO(), podSpec.Name, types.JSONPatchType,
		[]byte(`[{"op": "add", "path": "/metadata/labels/`+key+`", "value": "`+value+`"}]`), metav1.PatchOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to add label to pod %s/%s label: %s:%s", nsName, podSpec.Name, key, value)
	}
	return nil
}

// RemoveLabelFromPod removes a label frm a pod
func (k *KubeManager) RemoveLabelFromPod(podSpec *entities.Pod, key string) error {
	nsName := podSpec.Namespace

	_, err := k.clientSet.CoreV1().Pods(nsName).Patch(context.TODO(), podSpec.Name, types.JSONPatchType,
		[]byte(`[{"op": "remove", "path": "/metadata/labels/`+key+`"}]`), metav1.PatchOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to remove label from pod %s/%s label: %s:%s", nsName, podSpec.Name, key)
	}
	return nil
}

// DeletePod deletes pod from a namespace
func (k *KubeManager) DeletePod(podName, namespaceName string) error {
	err := k.clientSet.CoreV1().Pods(namespaceName).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to delete pod %s/%s", namespaceName, podName)
	}
	return nil
}

// CreateNamespace creates a new K8S namespace
func (k *KubeManager) CreateNamespace(nsSpec *v1.Namespace) (*v1.Namespace, error) {
	namespace, err := k.clientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update namespace %s", nsSpec.Name)
	}
	return namespace, nil
}

// DeleteNamespaces deletes all the namespaces in a list
func (k *KubeManager) DeleteNamespaces(namespaceNames []string) error {
	for _, ns := range namespaceNames {
		err := k.clientSet.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to delete namespace %s", ns)
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
	for _, node := range kubeNode.Items { // nolint
		for _, cond := range node.Status.Conditions {
			if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
				nodes = append(nodes, node.DeepCopy())
			}
		}
	}
	return nodes, nil
}

// GetPod gets a pod by namespace and name.
func (k *KubeManager) GetPod(ns, name string) (*v1.Pod, error) {
	kubePod, err := k.clientSet.CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get pod %s/%s", ns, name)
	}
	return kubePod, nil
}

// probeConnectivity execs into a pod and checks its connectivity to another pod..
func (k *KubeManager) ProbeConnectivity(nsFrom, podFrom, containerFrom, addrTo string, protocol v1.Protocol, toPort int) (bool, string, error) { // nolint
	var cmd []string
	port := strconv.Itoa(toPort)

	switch protocol {
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

// ProbeConnectivityWithCurl execs into a pod and connect the endpoint, return endpoint
func (k *KubeManager) ProbeConnectivityWithCurl(nsFrom, podFrom, containerFrom, addrTo string, protocol v1.Protocol, toPort int) (bool, string, string, error) { // nolint
	var cmd []string
	port := strconv.Itoa(toPort)

	cmd = []string{"/usr/bin/curl", "-g", "-q", "-s", "telnet://"+net.JoinHostPort(addrTo, port)}

	commandDebugString := fmt.Sprintf("kubectl exec %s -c %s -n %s -- %s", podFrom, containerFrom, nsFrom, strings.Join(cmd, " "))
	k.Logger.Debug("commandDebugString "+ commandDebugString)
	stdout, stderr, err := k.executeRemoteCommand(nsFrom, podFrom, containerFrom, cmd)
	if err != nil {
		fmt.Println(fmt.Printf("%s/%s -> %s: error when running command: err - %v /// stdout - %s /// stderr - %s", nsFrom, podFrom, addrTo, err, stdout, stderr))
		return false, "", commandDebugString, nil
	}
	ep := strings.TrimSpace(stdout)
	return true, ep, commandDebugString, nil
}

// executeRemoteCommand executes a remote shell command on the given pod.
func (k *KubeManager) executeRemoteCommand(namespace, pod, containerName string, command []string) (string, string, error) { // nolint
	return k8s.ExecWithOptions(k.config, k.clientSet, &k8s.ExecOptions{
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
	k.Logger.Info("Waiting for HTTP servers (ports 80 and 81) to become ready")

	testCases := map[string]*TestCase{}
	ports, protocols := model.AllPortsProtocol()

	for _, port := range ports {
		for _, protocol := range protocols {
			fromPort := 81
			desc := fmt.Sprintf("%d->%d,%s", fromPort, port, protocol)
			testCases[desc] = &TestCase{ToPort: int(port), Protocol: protocol, ServiceType: entities.PodIP}
		}
	}

	notReady := map[string]bool{}
	for caseName := range testCases {
		notReady[caseName] = true
	}

	const maxTries = 10
	for i := 0; i < maxTries; i++ {
		for caseName, testCase := range testCases {
			if !notReady[caseName] {
				continue
			}
			reachability := NewReachability(model.AllPods(), true)
			testCase.Reachability = reachability
			ProbePodToPodConnectivity(k, model, testCase, false)
			_, wrong, _, _ := reachability.Summary(false)
			if wrong == 0 {
				k.Logger.Info("Server is ready", zap.String("case", caseName))
				delete(notReady, caseName)
			} else {
				k.Logger.Info("Server is not ready", zap.String("case", caseName))
			}
		}
		if len(notReady) == 0 {
			return nil
		}
		time.Sleep(waitInterval)
	}
	return errors.Errorf("after %d tries, %d HTTP servers are not ready", maxTries, len(notReady))
}
