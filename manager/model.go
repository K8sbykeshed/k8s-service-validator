package manager

import (
	"github.com/k8sbykeshed/k8s-service-lb-validator/manager/data"
	v1 "k8s.io/api/core/v1"
)

type Model struct {
	Namespaces     []*data.Namespace
	allPodStrings  *[]data.PodString
	allPods        *[]*data.Pod
	NamespaceNames []string
	PodNames       []string
	Ports          []int32
	Protocols      []v1.Protocol
	DNSDomain      string
}

// AllPodStrings returns a slice of all pod strings
func (m *Model) AllPodStrings() []data.PodString {
	if m.allPodStrings == nil {
		var pods []data.PodString
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
		var pods []*data.Pod
		for _, podName := range podNames {
			var containers []*data.Container
			for _, port := range ports {
				for _, protocol := range protocols {
					containers = append(containers, &data.Container{
						Port:     port,
						Protocol: protocol,
					})
				}
			}
			pods = append(pods, &data.Pod{
				Namespace:  ns,
				Name:       podName,
				Containers: containers,
			})
		}
		model.Namespaces = append(model.Namespaces, &data.Namespace{Name: ns, Pods: pods})
	}
	return model
}

// AllPods returns a slice of all pods
func (m *Model) AllPods() []*data.Pod {
	if m.allPods == nil {
		var pods []*data.Pod
		for _, ns := range m.Namespaces {
			pods = append(pods, ns.Pods...)
		}
		m.allPods = &pods
	}
	return *m.allPods
}
