package manager

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Model struct {
	Namespaces    []*Namespace
	allPodStrings *[]PodString
	allPods       *[]*Pod
	// the raw data
	NamespaceNames []string
	PodNames       []string
	Ports          []int32
	Protocols      []v1.Protocol
	DNSDomain      string
}

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
		var pods []*Pod
		for _, podName := range podNames {
			var containers []*Container
			for _, port := range ports {
				for _, protocol := range protocols {
					containers = append(containers, &Container{
						Port:     port,
						Protocol: protocol,
					})
				}
			}
			pods = append(pods, &Pod{
				Namespace:  ns,
				Name:       podName,
				Containers: containers,
			})
		}
		model.Namespaces = append(model.Namespaces, &Namespace{Name: ns, Pods: pods})
	}
	return model
}

// AllPods returns a slice of all pods
func (m *Model) AllPods() []*Pod {
	if m.allPods == nil {
		var pods []*Pod
		for _, ns := range m.Namespaces {
			for _, pod := range ns.Pods {
				pods = append(pods, pod)
			}
		}
		m.allPods = &pods
	}
	return *m.allPods
}

// GetModel
func GetModel(cs *kubernetes.Clientset, config *rest.Config) (string, *Model, *KubeManager) {
	rootNs, domain := "name", "cluster.local"
	manager := NewKubeManager(cs, config)
	nsX, namespaces := getNamespaces(rootNs)
	model := NewModel(namespaces, []string{"a", "b", "c"}, []int32{80, 81}, []v1.Protocol{v1.ProtocolTCP}, domain)
	return nsX, model, manager
}

// getNamespaces
func getNamespaces(rootNs string) (string, []string) {
	rootNs += "-"
	nsX := fmt.Sprintf("%sx", rootNs)
	return nsX, []string{nsX}
}