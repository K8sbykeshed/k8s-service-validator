package manager

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	poll = 2 * time.Second
)

var errPodCompleted = fmt.Errorf("pod ran to completion")

// KubeManager is the core struct to manage kubernetes objects
type KubeManager struct {
	config    *rest.Config
	clientSet *kubernetes.Clientset
}

// NewKubeManager returns a new KubeManager
func NewKubeManager(cs *kubernetes.Clientset, config *rest.Config) *KubeManager {
	return &KubeManager{clientSet: cs, config: config}
}

// InitializeCluster start all pods and wait them to be up
func (k *KubeManager) InitializeCluster(model *Model) error {
	var createdPods []*v1.Pod
	for _, ns := range model.Namespaces { // create namespaces
		if _, err := k.CreateNamespace(ns.Spec()); err != nil {
			return err
		}
		for _, pod := range ns.Pods {
			fmt.Println(fmt.Printf("creating/updating pod %s/%s", ns.Name, pod.Name))

			kubePod, err := k.createPod(pod.KubePod())
			if err != nil {
				return err
			}
			createdPods = append(createdPods, kubePod)
		}
	}
	for _, createdPod := range createdPods {
		if err := WaitForPodRunningInNamespace(k.clientSet, createdPod); err != nil {
			return errors.Wrapf(err, "unable to wait for pod %s/%s", createdPod.Namespace, createdPod.Name)
		}
	}
	fmt.Println("all done")
	return nil
}

// createPod is a convenience function for pod setup.
func (k *KubeManager) createPod(pod *v1.Pod) (*v1.Pod, error) {
	ns := pod.Namespace
	fmt.Println(fmt.Printf("creating pod %s/%s", ns, pod.Name))
	createdPod, err := k.clientSet.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update pod %s/%s", ns, pod.Name)
	}
	return createdPod, nil
}

// CreateService is a convenience function for service setup.
func (k *KubeManager) CreateService(service *v1.Service) (*v1.Service, error) {
	ns := service.Namespace
	name := service.Name

	createdService, err := k.clientSet.CoreV1().Services(ns).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create service %s/%s", ns, name)
	}
	return createdService, nil
}

func (k *KubeManager) CreateNamespace(ns *v1.Namespace) (*v1.Namespace, error) {
	createdNamespace, err := k.clientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update namespace %s", ns.Name)
	}
	return createdNamespace, nil
}

func (k *KubeManager) DeleteNamespaces(namespaces []string) error {
	for _, ns := range namespaces {
		err := k.clientSet.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to delete namespace %s", ns)
		}
	}
	return nil
}

// probeConnectivity execs into a pod and checks its connectivity to another pod..
func (k *KubeManager) probeConnectivity(nsFrom string, podFrom string, containerFrom string, addrTo string, protocol v1.Protocol, toPort int) (bool, string, error) {
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
func (k *KubeManager) executeRemoteCommand(namespace string, pod string, containerName string, command []string) (string, string, error) {
	return ExecWithOptions(k.config, k.clientSet, ExecOptions{
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
