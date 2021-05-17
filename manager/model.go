package manager

import (
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/workload"
	v1 "k8s.io/api/core/v1"
)

type Model struct {
	Namespaces     []*workload.Namespace
	allPodStrings  *[]workload.PodString
	allPods        *[]*workload.Pod
	NamespaceNames []string
	PodNames       []string
	Ports          []int32
	Protocols      []v1.Protocol
	DNSDomain      string
}

// AllPodStrings returns a slice of all pod strings
func (m *Model) AllPodStrings() []workload.PodString {
	if m.allPodStrings == nil {
		var pods []workload.PodString
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
func NewModel(namespaces, podNames []string, ports []int32, protocols []v1.Protocol, dnsDomain string) *Model {
	model := &Model{
		NamespaceNames: namespaces,
		PodNames:       podNames,
		Ports:          ports,
		Protocols:      protocols,
		DNSDomain:      dnsDomain,
	}
	// build the entire "model" for the overall test, which means, building
	// namespaces, pods, containers for each protocol.
	for _, ns := range namespaces {
		var pods []*workload.Pod
		for _, podName := range podNames {
			var containers []*workload.Container
			for _, port := range ports {
				for _, protocol := range protocols {
					containers = append(containers, &workload.Container{
						Port:     port,
						Protocol: protocol,
					})
				}
			}
			pods = append(pods, &workload.Pod{
				Namespace:  ns,
				Name:       podName,
				Containers: containers,
			})
		}
		model.Namespaces = append(model.Namespaces, &workload.Namespace{Name: ns, Pods: pods})
	}
	return model
}

// AllPods returns a slice of all pods
func (m *Model) AllPods() []*workload.Pod {
	if m.allPods == nil {
		var pods []*workload.Pod
		for _, ns := range m.Namespaces {
			for _, pod := range ns.Pods {
				pods = append(pods, pod)
			}
		}
		m.allPods = &pods
	}
	return *m.allPods
}
