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
	GetLabel(string) (string, error)
	SetLabel(string, string) error
	RemoveLabel(string) error
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

// GetLabel returns the value of label specified with key. ErrLabelNotFound if not present.
func (s *Service) GetLabel(key string) (value string, err error) {
	labels := s.service.ObjectMeta.Labels
	if labels == nil {
		return "", ErrLabelNotFound
	}
	value, ok := labels[key]
	if !ok {
		return "", ErrLabelNotFound
	}
	return value, nil
}

// SetLabel updates the value of label specified by key. Else if key does not exist,
// create the key-value pair.
func (s *Service) SetLabel(key, value string) error {
	var err error
	if s.service, err = s.clientSet.CoreV1().Services(s.service.Namespace).Get(context.TODO(), s.service.Name, metav1.GetOptions{}); err != nil {
		return errors.Wrapf(err, "unable to get service %s", s.service.Name)
	}
	labels := s.service.ObjectMeta.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	s.service.ObjectMeta.Labels = labels
	if _, err = s.clientSet.CoreV1().Services(s.service.Namespace).Update(context.TODO(), s.service, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "unable to update service %s", s.service.Name)
	}
	return nil
}

// RemoveLabel removes the label specified by key. ErrLabelNotFound if not present.
func (s *Service) RemoveLabel(key string) error {
	var err error
	if s.service, err = s.clientSet.CoreV1().Services(s.service.Namespace).Get(context.TODO(), s.service.Name, metav1.GetOptions{}); err != nil {
		return errors.Wrapf(err, "unable to get service %s", s.service.Name)
	}
	labels := s.service.ObjectMeta.Labels
	if labels == nil {
		return ErrLabelNotFound
	}
	if _, ok := labels[key]; !ok {
		return ErrLabelNotFound
	}
	delete(labels, key)
	s.service.ObjectMeta.Labels = labels
	opts := metav1.UpdateOptions{}
	if _, err := s.clientSet.CoreV1().Services(s.service.Namespace).Update(context.TODO(), s.service, opts); err != nil {
		return errors.Wrapf(err, "unable to update service %s", s.service.Name)
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
			endpoint := event.Object.(*v1.Endpoints) // nolint
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

// WaitForClusterIP returns the NodePort number, by pausing the process until timeout or ClusterIP is created
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
			svc := event.Object.(*v1.Service) // nolint
			if svc.Name == s.service.Name && svc.Spec.ClusterIP != "" {
				return svc.Spec.ClusterIP, nil
			}
		case <-time.After(waitTime):
			return "", nil
		}
	}
}

// WaitForNodePort returns nodePort, by pausing the process until timeout nodePort is created
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
			svc := event.Object.(*v1.Service) // nolint
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
			svc := event.Object.(*v1.Service) // nolint
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