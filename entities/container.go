package entities

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/k8sbykeshed/k8s-service-validator/commands"
	v1 "k8s.io/api/core/v1"
)

type (
	ContainerImage   string
	ContainerCommand string
)

const (
	// NotSpecifiedImage is used when user does not provide an image
	NotSpecifiedImage ContainerImage = ""

	// AgnhostImage is the image reference for agnhost server
	AgnhostImage ContainerImage = "k8s.gcr.io/e2e-test-images/agnhost:2.31"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Container represents the container model
type Container struct {
	Name     string
	Image    ContainerImage
	Command  []string
	Protocol v1.Protocol
	Port     int32
}

// GetName returns the parsed container name
func (c *Container) GetName() string {
	rand.Seed(time.Now().UnixNano())
	if c.Name == "" {
		if c.Port == 0 {
			c.Name = fmt.Sprintf("cont-%d", rand.Intn(1e5))
		} else {
			c.Name = fmt.Sprintf("cont-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
		}
	}
	return c.Name
}

//
// PortName returns the parsed container port name
func (c *Container) PortName() string {
	if c.Port == 0 {
		return fmt.Sprintf("serve-%d", rand.Intn(1e5))
	}
	return fmt.Sprintf("serve-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
}

// ToK8SSpec returns the Kubernetes container specification
func (c *Container) ToK8SSpec() v1.Container {
	var (
		cmd   = c.Command
		name  = c.GetName()
		image = AgnhostImage
	)

	// use user image instead of agnhost if specified
	if c.Image != NotSpecifiedImage {
		image = c.Image
	}
	if len(cmd) == 0 {
		switch image {
		case AgnhostImage:
			agnHost := commands.NewAgnHostServer(int(c.Port), c.Protocol)
			cmd = agnHost.ServeCommand()
		default:
			zap.L().Error(fmt.Sprintf("self-provided image %s should have non-empty command", image))
		}
	}

	// it must have a container/port tuple per container
	container := v1.Container{
		Name:            name,
		ImagePullPolicy: v1.PullIfNotPresent,
		Image:           string(image),
		Command:         cmd,
	}
	if c.Port > 0 {
		container.Ports = []v1.ContainerPort{
			{
				ContainerPort: c.Port,
				Name:          c.PortName(),
				Protocol:      c.Protocol,
			},
		}
	}

	return container
}
