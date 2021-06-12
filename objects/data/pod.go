package data

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// Pod represents a Pod model
type Pod struct {
	Namespace  string
	Name       string
	Containers []*Container
	NodeName   string
	// todo(knabben) add a service data and move ports there.
	PodIP      string
	HostIP     string
	ExternalIP string
	ToPort     int32
}

func (p *Pod) GetToPort() int32 {
	return p.ToPort
}

func (p *Pod) SetToPort(toPort int32) {
	p.ToPort = toPort
}

func (p *Pod) GetHostIP() string {
	return p.HostIP
}

func (p *Pod) SetHostIP(hostIP string) {
	p.HostIP = hostIP
}

func (p *Pod) GetPodIP() string {
	return p.PodIP
}

func (p *Pod) SetPodIP(podIP string) {
	p.PodIP = podIP
}

func (p *Pod) GetExternalIP() string {
	return p.ExternalIP
}

func (p *Pod) SetExternalIP(externalIP string) {
	p.ExternalIP = externalIP
}

// PodString returns a corresponding pod string
func (p *Pod) PodString() PodString {
	return NewPodString(p.Namespace, p.Name)
}

// ServiceName returns the unqualified service name
func (p *Pod) ServiceName() string {
	return fmt.Sprintf("s-%s-%s", p.Namespace, p.Name)
}

// QualifiedServiceAddress returns the address that can be used to hit a service from
// any namespace in the cluster
func (p *Pod) QualifiedServiceAddress(dnsDomain string) string {
	return fmt.Sprintf("%s.%s.svc.%s", p.ServiceName(), p.Namespace, dnsDomain)
}

// ContainerSpecs builds kubernetes container specs for the pod
func (p *Pod) ContainerSpecs() []v1.Container {
	containers := make([]v1.Container, len(p.Containers))
	for i, cont := range p.Containers {
		containers[i] = cont.Spec()
	}
	return containers
}

func (p *Pod) labelSelectorKey() string {
	return "pod"
}

func (p *Pod) labelSelectorValue() string {
	return p.Name
}

// LabelSelector returns the default labels that should be placed on a pod/deployment
// in order for it to be uniquely selectable by label selectors
func (p *Pod) LabelSelector() map[string]string {
	return map[string]string{
		p.labelSelectorKey(): p.labelSelectorValue(),
	}
}

// KubePod returns the kube pod
func (p *Pod) KubePod() *v1.Pod {
	zero := int64(0)
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Labels:    p.LabelSelector(),
			Namespace: p.Namespace,
		},
		Spec: v1.PodSpec{
			NodeName:                      p.NodeName,
			Containers:                    p.ContainerSpecs(),
			TerminationGracePeriodSeconds: &zero,
		},
	}
}

// PodString is the representation of the Pod on a string
type PodString string

// NewPodString generates a new PodString from the pod from pod name and namespace
func NewPodString(namespace, podName string) PodString {
	return PodString(fmt.Sprintf("%s/%s", namespace, podName))
}

// Namespace extracts the namespace
func (pod PodString) Namespace() string {
	ns, _ := pod.split()
	return ns
}

func (pod PodString) split() (string, string) {
	pieces := strings.Split(string(pod), "/")
	if len(pieces) != 2 {
		fmt.Println(fmt.Printf("expected ns/pod, found %+v", pieces))
	}
	return pieces[0], pieces[1]
}

// String stringify the pod
func (pod PodString) String() string {
	return string(pod)
}

// PodName extracts the pod name
func (pod PodString) PodName() string {
	_, podName := pod.split()
	return podName
}
