package tools

import (
	"testing"

	"github.com/k8sbykeshed/k8s-service-validator/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-validator/matrix"
)

func ResetTestBoard(t *testing.T, s kubernetes.Services, m *matrix.Model) {
	if err := s.Delete(); err != nil {
		t.Fatal(err)
	}
	m.ResetAllPods()
}

func MustNoWrong(wrongNum int, t *testing.T) {
	if wrongNum > 0 {
		t.Errorf("Wrong results number %d", wrongNum)
	}
}
