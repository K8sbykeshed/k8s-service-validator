package matrix

import (
	"fmt"

	"github.com/k8sbykeshed/k8s-service-validator/entities"
	v1 "k8s.io/api/core/v1"
)

// Model defines the model for cluster data
type Model struct {
	pods      *[]*entities.Pod // cache for pods, automatically filled
	ports     *[]int32         // cache for ports, automatically filled
	protocols *[]v1.Protocol   // cache for protocols, automatically filled

	Namespaces []*entities.Namespace
	dnsDomain  string
}

// NewModel construct a Model struct used on probing and reachability comparison
func NewModel(namespaceNames, podNames []string, ports []int32, protocols []v1.Protocol,
	dnsDomain string) *Model {
	// build the entire "model" for the overall test, which means, building
	// namespaces, pods, containers for each protocol`.
	namespaces := make([]*entities.Namespace, len(namespaceNames))
	for i := range namespaces {
		ns := namespaceNames[i]
		namespaces[i] = entities.NewNamespaceWithPods(ns, podNames, ports, protocols)
	}
	return &Model{
		Namespaces: namespaces,
		dnsDomain:  dnsDomain,
	}
}

// AddNamespaceWithImageAndCommands creates a new namespace in the model with pods
// while explicitly specifying the container image and commands to use
func (m *Model) AddNamespaceWithImageAndCommands(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol,
	image entities.ContainerImage, commands []string) *entities.Namespace {
	namespace := entities.NewNamespaceWithPodsGivenImageAndCommand(namespaceName, podNames, ports, protocols, image, commands)
	m.Namespaces = append(m.Namespaces, namespace)
	return namespace
}

// AddNamespace creates a new namespace in the model with pods
func (m *Model) AddNamespace(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol) *entities.Namespace {
	namespace := entities.NewNamespaceWithPods(namespaceName, podNames, ports, protocols)
	m.Namespaces = append(m.Namespaces, namespace)
	m.pods = nil // repopulate the pod cache
	return namespace
}

// AddIPerfNamespace creates a new namespace in the model with pods
// but provides the iperf serve command
func (m *Model) AddIPerfNamespace(namespaceName string, podNames []string, ports []int32, protocols []v1.Protocol) *entities.Namespace {
	iperfNamespace := entities.NewNamespaceWithIPerfPods(namespaceName, podNames, ports, protocols)
	m.Namespaces = append(m.Namespaces, iperfNamespace)
	m.pods = nil // repopulate the pod cache
	return iperfNamespace
}

func extractPortProtocols(namespaces []*entities.Namespace) ([]int32, []v1.Protocol) {
	var (
		ports     []int32
		protocols []v1.Protocol
	)
	for _, ns := range namespaces {
		for _, pod := range ns.Pods {
			for _, container := range pod.Containers {
				if container.Port != 0 && !intOnSlice(container.Port, ports) {
					ports = append(ports, container.Port)
				}
				if container.Protocol != "" && !protocolOnSlice(container.Protocol, protocols) {
					protocols = append(protocols, container.Protocol)
				}
			}
		}
	}
	return ports, protocols
}

// NewModelWithNamespace returns a new Model based on existent namespaces
func NewModelWithNamespace(namespaces []*entities.Namespace, dnsDomain string) *Model {
	ports, protocols := extractPortProtocols(namespaces)
	return &Model{
		ports:      &ports,
		protocols:  &protocols,
		dnsDomain:  dnsDomain,
		Namespaces: namespaces,
	}
}

// AllPods returns a slice of all pods
func (m *Model) AllPods() []*entities.Pod {
	if m.pods == nil {
		var pods []*entities.Pod
		for _, ns := range m.Namespaces {
			pods = append(pods, ns.Pods...)
		}
		m.pods = &pods
	}
	return *m.pods
}

func (m *Model) ResetAllPods() {
	pods := m.AllPods()
	for _, p := range pods {
		p.Reset()
	}
}

// AllPortsProtocol returns a tuple of slices of all ports and protocols
func (m *Model) AllPortsProtocol() ([]int32, []v1.Protocol) {
	if m.ports == nil && m.protocols == nil {
		ports, protocols := extractPortProtocols(m.Namespaces)
		m.ports, m.protocols = &ports, &protocols
	}
	return *m.ports, *m.protocols
}

// AddPod adds pod into the cluster model
func (m *Model) AddPod(pod *entities.Pod, namespaceName string) {
	*m.pods = append(*m.pods, pod)

	for _, ns := range m.Namespaces {
		if ns.Name == namespaceName {
			ns.Pods = append(ns.Pods, pod)
		}
	}
}

// RemovePod removes pod from the cluster model
func (m *Model) RemovePod(podName, namespaceName string) error {
	foundNamespace := false
	var ns *entities.Namespace
	for _, n := range m.Namespaces {
		if n.Name == namespaceName {
			foundNamespace = true
			ns = n
		}
	}

	if foundNamespace {
		for i, p := range ns.Pods {
			if p.Name == podName {
				ns.Pods = append(ns.Pods[:i], ns.Pods[i+1:]...)
			}
		}
		for i, p := range *m.pods {
			if p.Name == podName {
				*m.pods = append((*m.pods)[:i], (*m.pods)[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("failed to find pod %s/%s", namespaceName, podName)
}
