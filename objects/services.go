package objects

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceBase interface {
	Create() v1.Service
	Verify()
	Status()
	Update()
}

type Service struct {
	service *v1.Service
	clientSet *kubernetes.Clientset
}

func NewService(client *kubernetes.Clientset, service *v1.Service) *Service {
	return &Service{
		service: service,
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
