package matrix

import (
	"github.com/k8sbykeshed/k8s-service-lb-validator/entities"
	v1 "k8s.io/api/core/v1"
)

// Model defines the model for cluster data
type Model struct {
	Namespaces     []*entities.Namespace
	allPodStrings  *[]entities.PodString
	allPods        *[]*entities.Pod
	NamespaceNames []string
	PodNames       []string
	Ports          []int32
	Protocols      []v1.Protocol
	DNSDomain      string
}

// AllPodStrings returns a slice of all pod strings
func (m *Model) AllPodStrings() []entities.PodString {
	if m.allPodStrings == nil {
		var pods []entities.PodString
		for _, ns := range m.Namespaces {
			for _, pod := range ns.Pods {
				pods = append(pods, pod.PodString())
			}
		}
		m.allPodStrings = &pods
	}
	return *m.allPodStrings
}

// NewModel returns the Model used to be probed
func NewModel(namespaceNames, podNames []string, ports []int32, protocols []v1.Protocol, dnsDomain string) *Model {
	var namespaces []*entities.Namespace

	// build the entire "model" for the overall test, which means, building
	// namespaces, pods, containers for each protocol.
	for _, ns := range namespaceNames {
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
		namespaces = append(namespaces, &entities.Namespace{Name: ns, Pods: pods})
	}
	return &Model{
		PodNames:       podNames,
		Ports:          ports,
		Protocols:      protocols,
		DNSDomain:      dnsDomain,
		Namespaces:     namespaces,
		NamespaceNames: namespaceNames,
	}
}

// AllPods returns a slice of all pods
func (m *Model) AllPods() []*entities.Pod {
	if m.allPods == nil {
		var pods []*entities.Pod
		for _, ns := range m.Namespaces {
			pods = append(pods, ns.Pods...)
		}
		m.allPods = &pods
	}
	return *m.allPods
}
