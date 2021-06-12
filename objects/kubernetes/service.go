package kubernetes

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

const (
	waitTime = 3 * time.Second
)

// ServiceBase contains the abstract implementation required for a service.
type ServiceBase interface {
	Create() (*v1.Service, error)
	Delete() error
	WaitForNodePort() (int32, error)
	WaitForEndpoint() (bool, error)
	WaitForExternalIP() ([]string, error)
}

type Services []*Service

// Delete delete all services in the list
func (s Services) Delete() error {
	for _, svc := range s {
		if err := svc.Delete(); err != nil {
			return err
		}
	}
	return nil
}

type Service struct {
	service   *v1.Service
	clientSet *kubernetes.Clientset
}

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
			endpoint := event.Object.(*v1.Endpoints)
			for _, subset := range endpoint.Subsets {
				if len(subset.Addresses) > 0 && len(subset.NotReadyAddresses) == 0 {
					return true, nil
				}
			}
		}
	}
	return false, nil
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
			svc := event.Object.(*v1.Service)
			if svc.Name == s.service.Name {
				for _, port := range svc.Spec.Ports {
					if port.NodePort != 0 {
						nodePort = port.NodePort
					}
				}
			}
		case <-time.After(waitTime):
			break
		}
	}
	return nodePort, nil
}

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
			svc := event.Object.(*v1.Service)
			if svc.Name == s.service.Name {
				for _, ip := range svc.Status.LoadBalancer.Ingress {
					ips = append(ips, ip.IP)
				}
			}
		case <-time.After(waitTime):
			break
		}
	}
	return ips, nil
}
