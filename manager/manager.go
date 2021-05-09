package manager

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	poll = 2 * time.Second
	waitInterval = 1 * time.Second
	waitTimeout  = 30 * time.Second
)

// KubeManager
type KubeManager struct {
	config *rest.Config
	clientSet *kubernetes.Clientset
}

// NewKubeManager
func NewKubeManager(cs *kubernetes.Clientset, config *rest.Config) *KubeManager {
	return &KubeManager{clientSet: cs, config: config}
}

// InitializeCluster
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

// WaitTimeoutForPodReadyInNamespace waits the given timeout duration for the
// specified pod to be ready and running.
func WaitForPodRunningInNamespace(c *kubernetes.Clientset, pod *v1.Pod) error {
	if pod.Status.Phase == v1.PodRunning {
		return nil
	}
	timeout := 5 * time.Minute
	return wait.PollImmediate(poll, timeout, podRunning(c, pod.Name, pod.Namespace))
}

var errPodCompleted = fmt.Errorf("pod ran to completion")

func podRunning(c *kubernetes.Clientset, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, errPodCompleted
		}
		return false, nil
	}
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

// createService is a convenience function for service setup.
func (k *KubeManager) CreateService(service *v1.Service) (*v1.Service, error) {
	ns := service.Namespace
	name := service.Name

	createdService, err := k.clientSet.CoreV1().Services(ns).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create service %s/%s", ns, name)
	}
	return createdService, nil
}

// CreateNamespace
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


// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command       []string
	Namespace     string
	PodName       string
	ContainerName string
	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
	Quiet              bool
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

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(config *rest.Config, cs *kubernetes.Clientset, options ExecOptions) (string, string, error) {
	var tty = false
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), config, options.Stdin, &stdout, &stderr, tty)
	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
