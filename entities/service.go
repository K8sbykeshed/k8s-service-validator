package entities

import (
	"fmt"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constants for services
const (
	PodIP        = "podip"
	ClusterIP    = "clusteip"
	NodePort     = "nodeport"
	ServiceName  = "servicename"
	LoadBalancer = "loadbalancer"

	Allprotocols = "allprotocols"
)

type Service struct {
	Name      string
	Namespace string
	Selector  map[string]string
}

// serviceID prevent conflicts when creating multiple services for same pod
var svcID *ServiceID

type ServiceID struct {
	mu sync.Mutex
	ID int
}

// NewService returns the service boilerplate
func NewService(p *Pod) *v1.Service {
	IncreaseServiceID()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", p.ServiceName(), svcID.ID),
			Namespace: p.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: p.LabelSelector(),
		},
	}
}

// NewService returns the service boilerplate based on service template
func NewServiceFromTemplate(t Service) *v1.Service {
	IncreaseServiceID()
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", t.Name, svcID.ID),
			Namespace: t.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: t.Selector,
		},
	}
}

// portFromContainer is a helper to return port spec from the service
func portFromContainer(containers []*Container, protocol v1.Protocol) []v1.ServicePort {
	portsSet := map[v1.ServicePort]bool{}
	for _, container := range containers {
		if protocol != Allprotocols && protocol != container.Protocol {
			continue
		}
		sp := v1.ServicePort{
			Name:     fmt.Sprintf("service-port-%s-%d", strings.ToLower(string(container.Protocol)), container.Port),
			Protocol: container.Protocol,
			Port:     container.Port,
		}
		portsSet[sp] = true
	}

	var ports []v1.ServicePort // nolint
	for p := range portsSet {
		ports = append(ports, p)
	}
	return ports
}

// ClusterIPService returns a kube service spec
func (p *Pod) ClusterIPService() *v1.Service {
	service := NewService(p)
	service.Spec.Ports = portFromContainer(p.Containers, Allprotocols)
	return service
}

// NodePortService returns a new node port service
func (p *Pod) NodePortService() *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeNodePort
	service.Spec.Ports = portFromContainer(p.Containers, Allprotocols)
	return service
}

// ExternalNameService returns a new external name service
func (p *Pod) ExternalNameService(domain string) *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeExternalName
	service.Spec.ExternalName = domain
	return service
}

// LoadBalancerServiceByProtocol returns a new Load balancer service based on protocol
func (p *Pod) LoadBalancerServiceByProtocol(protocol v1.Protocol) *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeLoadBalancer
	service.Spec.Ports = portFromContainer(p.Containers, protocol)
	return service
}

// NodePortLocalService returns a new Load balancer service with local service.
// service.spec.externalTrafficPolicy - Local preserves the client source IP and avoids a second hop for
// LoadBalancer and NodePort type services, but risks potentially imbalanced traffic spreading.
func (p *Pod) NodePortLocalService() *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeNodePort
	service.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeLocal
	service.Spec.Ports = portFromContainer(p.Containers, Allprotocols)
	return service
}

func IncreaseServiceID() {
	if svcID == nil {
		svcID = &ServiceID{}
	}
	svcID.mu.Lock()
	defer svcID.mu.Unlock()
	svcID.ID++
}
