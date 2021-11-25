package matrix

import (
	"fmt"

	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
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
func NewModel(namespaceNames, podNames []string, ports []int32, protocols []v1.Protocol, dnsDomain string) *Model {
	// build the entire "model" for the overall test, which means, building
	// namespaces, pods, containers for each protocol`.

	namespaces := make([]*entities.Namespace, len(namespaceNames))
	for i := range namespaces {
		ns := namespaceNames[i]
		var pods []*entities.Pod
		for _, podName := range podNames {
			var containers []*entities.Container
			for _, port := range ports {
				for _, protocol := range protocols {
					containers = append(containers, &entities.Container{
						Port:     port,
						Protocol: protocol,
					})
				}
			}
			pods = append(pods, &entities.Pod{Namespace: ns, Name: podName, Containers: containers})
		}
		namespaces[i] = &entities.Namespace{Name: ns, Pods: pods}
	}
	return &Model{
		Namespaces: namespaces,
		dnsDomain:  dnsDomain,
	}
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
	for i, n := range m.Namespaces {
		if n.Name == namespaceName {
			m.Namespaces = append(m.Namespaces[:i], m.Namespaces[i+1:]...)
			foundNamespace = true
		}
	}

	if foundNamespace {
		for i, p := range *m.pods {
			if p.Name == podName {
				*m.pods = append((*m.pods)[:i], (*m.pods)[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("failed to find pod %s/%s", namespaceName, podName)
}
