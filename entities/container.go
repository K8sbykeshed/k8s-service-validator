package entities

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const AgnhostImage = "k8s.gcr.io/e2e-test-images/agnhost:2.31"

// Container represents the container model
type Container struct {
	Command  []string
	Protocol v1.Protocol
	Port     int32
}

// NewContainer returns a new internal container
func NewContainer(port int32, protocol v1.Protocol, cmd []string) *Container {
	return &Container{
		Command:  cmd,
		Protocol: protocol,
		Port:     port,
	}
}

// Name returns the parsed container name
func (c *Container) Name() string {
	return fmt.Sprintf("cont-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
}

//
// PortName returns the parsed container port name
func (c *Container) PortName() string {
	return fmt.Sprintf("serve-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
}

// ToK8SSpec returns the Kubernetes container specification
func (c *Container) ToK8SSpec() v1.Container {
	var cmd = c.Command

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

	// it must have a container/port tuple per container
	return v1.Container{
		Name:            c.Name(),
		Image:           AgnhostImage,
		ImagePullPolicy: v1.PullIfNotPresent,
		Ports: []v1.ContainerPort{
			{
				ContainerPort: c.Port,
				Name:          c.PortName(),
				Protocol:      c.Protocol,
			},
		},
		Command: cmd,
	}
}
