package matrix

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	ek "github.com/k8sbykeshed/k8s-service-validator/entities/kubernetes"
)

const (
	waitInterval = 1 * time.Second
)

// KubeManager is the core struct to manage kubernetes entities
type KubeManager struct {
	config    *rest.Config
	clientSet *kubernetes.Clientset
}

// NewKubeManager returns a new KubeManager
func NewKubeManager(cs *kubernetes.Clientset, config *rest.Config) *KubeManager {
	return &KubeManager{clientSet: cs, config: config}
}

// GetClientSet returns the Kubernetes clientset
func (k *KubeManager) GetClientSet() *kubernetes.Clientset {
	return k.clientSet
}

// StartPods start all pods and wait them to be up
func (k *KubeManager) StartPods(model *Model, nodes []*v1.Node) error {
	zap.L().Info("Creating test pods in the cluster.")
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
			zap.L().Debug("creating/updating pod.",
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
	zap.L().Debug("Wait for pod running.", zap.String("name", modelPod.Name), zap.String("namespace", modelPod.Namespace))

	if err := ek.WaitForPodRunningInNamespace(k.clientSet, kubePod); err != nil {
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
func (k *KubeManager) AddLabelToPod(podSpec *entities.Pod, key, value string) error {
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
		return errors.Wrapf(err, "unable to remove label from pod %s/%s label key: %s", nsName, podSpec.Name, key)
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
		zap.L().Error(fmt.Sprintf("protocol %s not supported", protocol))
	}

	commandDebugString := fmt.Sprintf("kubectl exec %s -c %s -n %s -- %s", podFrom, containerFrom, nsFrom, strings.Join(cmd, " "))
	_, stderr, err := k.executeRemoteCommand(nsFrom, podFrom, containerFrom, cmd)
	zap.L().Error(
		fmt.Sprintf("Can't connect: %s/%s -> %s", nsFrom, podFrom, addrTo),
		zap.String("stderr", stderr), zap.Error(err),
	)
	if err != nil {
		return false, commandDebugString, nil
	}
	return true, commandDebugString, nil
}

// ProbeConnectivityWithCurl execs into a pod and connect the endpoint, return endpoint
func (k *KubeManager) ProbeConnectivityWithNc(nsFrom, podFrom, containerFrom, addrTo string, protocol v1.Protocol, toPort int) (bool, string, string, error) { // nolint
	var cmd []string
	var err error
	port := strconv.Itoa(toPort)

	switch protocol {
	case v1.ProtocolTCP:
		cmd = []string{"nc", "-w10", addrTo, port}
	case v1.ProtocolUDP:
		cmd = []string{"nc", "-u", "-w10", addrTo, port}
	default:
		zap.L().Error(fmt.Sprintf("protocol %s not supported", protocol))
	}

	commandDebugString := fmt.Sprintf("kubectl exec %s -c %s -n %s -- %s", podFrom, containerFrom, nsFrom, strings.Join(cmd, " "))
	zap.L().Debug("commandDebugString " + commandDebugString)

	maxRetries := 3
	var stdout string
	for i := 0; i < maxRetries; i++ {
		stdout, _, err := k.executeRemoteCommand(nsFrom, podFrom, containerFrom, cmd)
		if err == nil {
			ep := strings.TrimSpace(stdout)
			return true, ep, commandDebugString, nil
		}
	}
	return false, "", commandDebugString, errors.Wrapf(err, fmt.Sprintf("%s/%s -> %s: error when running command:"+
		" err - %v /// stdout - %s", nsFrom, podFrom, addrTo, err, stdout))
}

// executeRemoteCommand executes a remote shell command on the given pod.
func (k *KubeManager) executeRemoteCommand(namespace, pod, containerName string, command []string) (string, string, error) { // nolint
	return ek.ExecWithOptions(k.config, k.clientSet, &ek.ExecOptions{
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
	zap.L().Info("Waiting for HTTP servers (ports 80 and 81) to become ready")

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
				zap.L().Debug("Server is ready", zap.String("case", caseName))
				delete(notReady, caseName)
			} else {
				zap.L().Error("Server is not ready", zap.String("case", caseName))
			}
		}
		if len(notReady) == 0 {
			zap.L().Info("Pods are ready, starting the test suite.")
			return nil
		}
		time.Sleep(waitInterval)
	}
	return errors.Errorf("after %d tries, %d HTTP servers are not ready", maxTries, len(notReady))
}

// CreateServiceFromTemplate creates k8s service based on template
func CreateServiceFromTemplate(cs *kubernetes.Clientset, t entities.ServiceTemplate) (string, ek.ServiceBase, string, error) { //nolint
	entities.IncreaseServiceID()

	servicePorts := make([]v1.ServicePort, len(t.ProtocolPorts))
	for i, sp := range t.ProtocolPorts {
		servicePorts[i] = v1.ServicePort{
			Name:     fmt.Sprintf("service-port-%s-%v", strings.ToLower(string(sp.Protocol)), sp.Port),
			Protocol: sp.Protocol,
			Port:     sp.Port,
		}
	}

	s := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", t.Name, entities.SvcID.ID),
			Namespace: t.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: t.Selector,
			Ports:    servicePorts,
		},
	}
	if t.SessionAffinity {
		s.Spec.SessionAffinity = "ClientIP"
	}

	var service ek.ServiceBase = ek.NewService(cs, s)
	if _, err := service.Create(); err != nil {
		return "", nil, "", errors.Wrapf(err, "failed to create service")
	}

	// wait for final status
	clusterIP, err := service.WaitForClusterIP()
	if err != nil || clusterIP == "" {
		return "", nil, "", errors.Wrapf(err, "no cluster IP available")
	}
	return s.Name, service, clusterIP, nil
}

func (k *KubeManager) InitializePod(pod *entities.Pod) error {
	if _, err := k.CreatePod(pod.ToK8SSpec()); err != nil {
		return err
	}
	if err := k.WaitAndSetIPs(pod); err != nil {
		return err
	}

	return nil
}
