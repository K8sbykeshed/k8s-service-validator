package entities

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k8sbykeshed/k8s-service-validator/pkg/commands"
)

const IPerfNamespaceSuffix = "-iperf"

// Namespace defines the structure of namespace
type Namespace struct {
	Name string
	Pods []*Pod
}

// Spec constructs k8s namespace
func (ns *Namespace) Spec() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ns.Name,
			Labels: ns.LabelSelector(),
		},
	}
}

// LabelSelector returns label selector fot the namespace
func (ns *Namespace) LabelSelector() map[string]string {
	return map[string]string{"ns": ns.Name}
}

// NewNamespaceWithPodsGivenImageAndCommand creates a new namespace given a combinations of pod names, ports and protocol
// also the container image and command are specified
func NewNamespaceWithPodsGivenImageAndCommand(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol,
	image ContainerImage, command []string,
) *Namespace {
	pods := make([]*Pod, 0)
	for _, podName := range podNames {
		var containers []*Container
		for _, port := range ports {
			for _, protocol := range protocols {
				containers = append(containers, &Container{
					Port:     port,
					Protocol: protocol,
					Image:    image,
					Command:  command,
				})
			}
		}
		pods = append(pods, &Pod{Namespace: namespaceName, Name: podName, Containers: containers})
	}
	return &Namespace{Name: namespaceName, Pods: pods}
}

// NewNamespaceWithIPerfPods creates a new namespace given a combinations of pod names, ports and protocol
// it uses the agnhost image but the iperf serve command
func NewNamespaceWithIPerfPods(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol) *Namespace {
	pods := make([]*Pod, 0)
	for _, podName := range podNames {
		var containers []*Container
		for _, port := range ports {
			for _, protocol := range protocols {
				iperf := commands.NewIPerfServer(int(port), protocol)
				containers = append(containers, &Container{
					Port:     port,
					Protocol: protocol,
					Image:    AgnhostImage,
					Command:  iperf.ServeCommand(),
				})
			}
		}
		pods = append(pods, &Pod{Namespace: namespaceName, Name: podName, Containers: containers})
	}
	return &Namespace{Name: namespaceName, Pods: pods}
}

// NewNamespaceWithPods creates a new namespace given a combinations of pod names, ports and protocol
// without explicitly specifies the container image
// we use agnhost serve hostname in this case
func NewNamespaceWithPods(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol) *Namespace {
	return NewNamespaceWithPodsGivenImageAndCommand(namespaceName, podNames, ports, protocols, NotSpecifiedImage, []string{})
}
