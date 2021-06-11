package data

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodIP        = "podip"
	ClusterIP    = "clusteip"
	NodePort     = "nodeport"
	ExternalName = "externalname"
	LoadBalancer = "loadbalancer"
)

// NewService returns the service boilerplate
func NewService(p *Pod) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.ServiceName(),
			Namespace: p.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: p.LabelSelector(),
		},
	}
}

// portContainer is a helper to return port spec from the service
func portFromContainer(containers []*Container) []v1.ServicePort {
	servicesPort := make([]v1.ServicePort, len(containers))
	for i, container := range containers {
		sp := v1.ServicePort{
			Name:     fmt.Sprintf("service-port-%s-%d", strings.ToLower(string(container.Protocol)), container.Port),
			Protocol: container.Protocol,
			Port:     container.Port,
		}
		servicesPort[i] = sp
	}
	return servicesPort
}

// ClusterIPService returns a kube service spec
func (p *Pod) ClusterIPService() *v1.Service {
	service := NewService(p)
	service.Spec.Ports = portFromContainer(p.Containers)
	return service
}

// NodePortService returns a new node port service.
func (p *Pod) NodePortService() *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeNodePort
	service.Spec.Ports = portFromContainer(p.Containers)
	return service
}

// ExternalNameService returns a new external name service.
func (p *Pod) ExternalNameService(domain string) *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeExternalName
	service.Spec.ExternalName = domain
	return service
}

// LoadBalancerService returns a new Load balancer service.
func (p *Pod) LoadBalancerService() *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeLoadBalancer
	service.Spec.Ports = portFromContainer(p.Containers)
	return service
}

// NodePortLocalService returns a new Load balancer service with local service.
// service.spec.externalTrafficPolicy - Local preserves the client source IP and avoids a second hop for
// LoadBalancer and NodePort type services, but risks potentially imbalanced traffic spreading.
func (p *Pod) NodePortLocalService() *v1.Service {
	service := NewService(p)
	service.Spec.Type = v1.ServiceTypeNodePort
	service.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeLocal
	service.Spec.Ports = portFromContainer(p.Containers)
	return service
}
