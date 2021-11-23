package kubernetes

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	waitTime = 15 * time.Second
)

// ServiceBase contains the abstract implementation required for a service.
type ServiceBase interface {
	Create() (*v1.Service, error)
	Delete() error
	GetClusterIP() string
	WaitForClusterIP() (string, error)
	WaitForNodePort() (int32, error)
	WaitForEndpoint() (bool, error)
	WaitForExternalIP() ([]string, error)
}

// Services defines an array of Service
type Services []*Service

// Delete delete all services in the list
func (s Services) Delete() error {
	for _, svc := range s {
		if svc != nil {
			if err := svc.Delete(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Service defines the structure of a service
type Service struct {
	service   *v1.Service
	clientSet *kubernetes.Clientset
}

// NewService constructs a Service
func NewService(client *kubernetes.Clientset, service *v1.Service) *Service {
	return &Service{
		service:   service,
		clientSet: client,
	}
}

// Create a new service
func (s *Service) Create() (*v1.Service, error) {
	opts := metav1.CreateOptions{}
	createdSvc, err := s.clientSet.CoreV1().Services(s.service.Namespace).Create(context.TODO(), s.service, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to create service %s/%s", s.service.Name, s.service.Namespace)
	}
	return createdSvc, nil
}

// Delete an existent service
func (s *Service) Delete() error {
	opts := metav1.DeleteOptions{}
	err := s.clientSet.CoreV1().Services(s.service.Namespace).Delete(context.TODO(), s.service.Name, opts)
	if err != nil {
		return errors.Wrapf(err, "unable to delete service %s", s.service.Name)
	}
	return nil
}

// GetClusterIP returns the clusterIP from spec
func (s *Service) GetClusterIP() string {
	return s.service.Spec.ClusterIP
}

// WaitForEndpoint return when all addresses are ready.
func (s *Service) WaitForEndpoint() (bool, error) {
	opts := metav1.ListOptions{}
	endpointWatch, err := s.clientSet.CoreV1().Endpoints(s.service.Namespace).Watch(context.TODO(), opts)
	if err != nil {
		return false, err
	}

	for {
		select {
		case event := <-endpointWatch.ResultChan():
			endpoint := event.Object.(*v1.Endpoints)  // nolint
			for _, subset := range endpoint.Subsets {
				if len(subset.Addresses) > 0 && len(subset.NotReadyAddresses) == 0 {
					return true, nil
				}
			}
		case <-time.After(waitTime):
			return false, nil
		}
	}
}

// WaitForClusterIP returns the NodePort number, by pausing the process until time out or NodePort is created
func (s *Service) WaitForClusterIP() (string, error) {
	opts := metav1.ListOptions{}
	serviceWatch, err := s.clientSet.CoreV1().Services(s.service.Namespace).Watch(context.TODO(), opts)
	defer serviceWatch.Stop()
	if err != nil {
		return "", err
	}

	for {
		select {
		case event := <-serviceWatch.ResultChan():
			svc := event.Object.(*v1.Service)  // nolint
			if svc.Name == s.service.Name && svc.Spec.ClusterIP != "" {
				return svc.Spec.ClusterIP, nil
			}
		case <-time.After(waitTime):
			return "", nil
		}
	}
}

func (s *Service) WaitForNodePort() (int32, error) {
	var nodePort int32
	opts := metav1.ListOptions{}
	serviceWatch, err := s.clientSet.CoreV1().Services(s.service.Namespace).Watch(context.TODO(), opts)
	defer serviceWatch.Stop()
	if err != nil {
		return 0, err
	}

	for {
		select {
		case event := <-serviceWatch.ResultChan():
			svc := event.Object.(*v1.Service)  // nolint
			if svc.Name == s.service.Name {
				for _, port := range svc.Spec.Ports {
					if port.NodePort != 0 {
						nodePort = port.NodePort
						return nodePort, nil
					}
				}
			}
		case <-time.After(waitTime):
			return nodePort, nil
		}
	}
}

// WaitForExternalIP pause the process until timeout and returns the array of ExteranlIP
func (s *Service) WaitForExternalIP() ([]string, error) {
	var ips []string
	opts := metav1.ListOptions{}
	serviceWatch, err := s.clientSet.CoreV1().Services(s.service.Namespace).Watch(context.TODO(), opts)
	if err != nil {
		return []string{}, nil
	}
	for {
		select {
		case event := <-serviceWatch.ResultChan():
			svc := event.Object.(*v1.Service)  // nolint
			if svc.Name == s.service.Name {
				for _, ip := range svc.Status.LoadBalancer.Ingress {
					ips = append(ips, ip.IP)
				}
			}
		case <-time.After(waitTime):
			return ips, nil
		}
	}
}
