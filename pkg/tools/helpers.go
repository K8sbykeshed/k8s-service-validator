package tools

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/pkg/errors"

	"github.com/k8sbykeshed/k8s-service-validator/pkg/entities/kubernetes"
	"github.com/k8sbykeshed/k8s-service-validator/pkg/matrix"
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

func RunCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)

	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to run cmd: %v", cmd.String()))
	}

	return out, nil
}
