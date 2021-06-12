package data

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strings"
)

const (
	agnHostImage = "k8s.gcr.io/e2e-test-images/agnhost:2.31"
)

// Container represents the container data model
type Container struct {
	Port     int32
	Protocol v1.Protocol
}

// Name returns the parsed container name
func (c *Container) Name() string {
	return fmt.Sprintf("cont-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
}

// PortName returns the parsed container port name
func (c *Container) PortName() string {
	return fmt.Sprintf("serve-%d-%s", c.Port, strings.ToLower(string(c.Protocol)))
}

// Spec returns the Kubernetes container specification
func (c *Container) Spec() v1.Container {
	var (
		agnHostImage = agnHostImage
		env          []v1.EnvVar
		cmd          []string
	)

	switch c.Protocol {
	case v1.ProtocolTCP:
		cmd = []string{"/agnhost", "serve-hostname", "--tcp", "--http=false", "--port", fmt.Sprintf("%d", c.Port)}
	case v1.ProtocolUDP:
		cmd = []string{"/agnhost", "serve-hostname", "--udp", "--http=false", "--port", fmt.Sprintf("%d", c.Port)}
	default:
		fmt.Println(fmt.Printf("invalid protocol %v", c.Protocol))
	}

	return v1.Container{
		Name:            c.Name(),
		ImagePullPolicy: v1.PullIfNotPresent,
		Image:           agnHostImage,
		Command:         cmd,
		Env:             env,
		SecurityContext: &v1.SecurityContext{},
		Ports: []v1.ContainerPort{
			{
				ContainerPort: c.Port,
				Name:          c.PortName(),
				Protocol:      c.Protocol,
			},
		},
	}
}
