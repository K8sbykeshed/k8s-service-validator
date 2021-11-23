package entities

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
)

// AgnhostImage is the image reference
const AgnhostImage = "k8s.gcr.io/e2e-test-images/agnhost:2.31"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Container represents the container model
type Container struct {
	Name     string
	Image    string
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
	if len(cmd) == 0 {
		switch c.Protocol {
		case v1.ProtocolTCP:
			cmd = []string{"/agnhost", "serve-hostname", "--tcp", "--http=false", "--port", fmt.Sprintf("%d", c.Port)}
		case v1.ProtocolUDP:
			cmd = []string{"/agnhost", "serve-hostname", "--udp", "--http=false", "--port", fmt.Sprintf("%d", c.Port)}
		default:
			fmt.Println(fmt.Printf("invalid protocol %v", c.Protocol))
		}
	}
	if c.Image != "" {
		image = c.Image
	}
	// it must have a container/port tuple per container
	container := v1.Container{
		Name:            name,
		ImagePullPolicy: v1.PullIfNotPresent,
		Image:           image,
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
