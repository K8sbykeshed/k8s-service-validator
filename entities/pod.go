package entities

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pod represents a Pod in the model view
type Pod struct {
	Namespace      string
	Name           string
	InitContainers []*Container
	Containers     []*Container
	NodeName       string
	// todo(knabben) add a service data and move ports there.
	PodIP       string
	HostIP      string
	ExternalIPs []ExternalIP
	ToPort      int32
}

// ExternalIP defines the struct of pod's external IP, which can be used to access from outside of node
type ExternalIP struct {
	IP       string
	Protocol v1.Protocol
}

// NewExternalIP creates ExternalIP based on ip address and protocol
func NewExternalIP(ip string, protocol v1.Protocol) ExternalIP {
	return ExternalIP{IP: ip, Protocol: protocol}
}

// NewExternalIPs creates array of ExternalIP based on array of IP addresses which share same protocol
func NewExternalIPs(ips []string, protocol v1.Protocol) []ExternalIP {
	var externalIPs []ExternalIP
	for _, ip := range ips {
		externalIPs = append(externalIPs, NewExternalIP(ip, protocol))
	}
	return externalIPs
}

// GetToPort returns the ToPort for the pod, which used to access the pod
func (p *Pod) GetToPort() int32 {
	return p.ToPort
}

// SetToPort sets the ToPort for the pod
func (p *Pod) SetToPort(toPort int32) {
	p.ToPort = toPort
}

// GetHostIP returns the HoseIP of the pod
func (p *Pod) GetHostIP() string {
	return p.HostIP
}

// SetHostIP sets the HostIP for the pod
func (p *Pod) SetHostIP(hostIP string) {
	p.HostIP = hostIP
}

// GetPodIP returns PodIP for the pod
func (p *Pod) GetPodIP() string {
	return p.PodIP
}

// SetPodIP sets the PodIP for the pod
func (p *Pod) SetPodIP(podIP string) {
	p.PodIP = podIP
}

// GetExternalIPs returns the array of ExternalIP for the pod
func (p *Pod) GetExternalIPs() []ExternalIP {
	return p.ExternalIPs
}

// GetExternalIPsByProtocol returns the ExternalIPs of the pod with the desired protocol ports
func (p *Pod) GetExternalIPsByProtocol(protocol v1.Protocol) []ExternalIP {
	var ips []ExternalIP
	for _, ip := range p.ExternalIPs {
		if ip.Protocol == protocol {
			ips = append(ips, ip)
		}
	}
	return ips
}

// SetExternalIPs sets the array of ExternalIPs for the pod
func (p *Pod) SetExternalIPs(externalIPs []ExternalIP) {
	p.ExternalIPs = externalIPs
}

// PodString returns a corresponding pod string
func (p *Pod) PodString() PodString {
	return NewPodString(p.Namespace, p.Name)
}

// ServiceName returns the unqualified service name
func (p *Pod) ServiceName() string {
	return fmt.Sprintf("s-%s-%s", p.Namespace, p.Name)
}

// QualifiedServiceAddress returns the address that can be used to hit a service from any namespace in the cluster
func (p *Pod) QualifiedServiceAddress(dnsDomain string) string {
	return fmt.Sprintf("%s.%s.svc.%s", p.ServiceName(), p.Namespace, dnsDomain)
}

// ContainersToK8SSpec builds kubernetes Containers specs for the pod
func ContainersToK8SSpec(cntrs []*Container) []v1.Container {
	k8sCntrs := make([]v1.Container, len(cntrs))
	for i, container := range cntrs {
		k8sCntrs[i] = container.ToK8SSpec()
	}
	return k8sCntrs
}

// LabelSelector returns the default labels that should be placed on a pod/deployment
// in order for it to be uniquely selectable by label selectors
func (p *Pod) LabelSelector() map[string]string {
	return map[string]string{"pod": p.Name}
}

// ToK8SSpec returns the Kubernetes pod specification
func (p *Pod) ToK8SSpec() *v1.Pod {
	zero := int64(0)
	podSpec := v1.PodSpec{
		NodeName:                      p.NodeName,
		Containers:                    ContainersToK8SSpec(p.Containers),
		TerminationGracePeriodSeconds: &zero,
	}
	if p.InitContainers != nil {
		podSpec.InitContainers = ContainersToK8SSpec(p.InitContainers)
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels:    p.LabelSelector(),
		},
		Spec: podSpec,
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
