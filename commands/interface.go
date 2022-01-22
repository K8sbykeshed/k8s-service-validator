package commands

import (
	"fmt"
	"strings"

	ek "github.com/k8sbykeshed/k8s-service-validator/entities/kubernetes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Executor supports Execute function that performs a probe, client method only
type Executor interface {
	Execute(config *rest.Config, cs *kubernetes.Clientset) (stdout, stderr string, err error)
}

// Client defines the methods supported by client-side command
type Client interface {
	DebugString() string
	ConnectCommand() []string
	Executor
}

// Server defines the methods supported by server-side command
type Server interface {
	ServeCommand() []string
}

// commandImpl keeps track of configurations of each type of commands
type commandImpl struct {
	nsFrom        string
	podFrom       string
	containerFrom string
	addrTo        string
	port          string
	protocol      v1.Protocol

	cmd []string
	Client
}

// DebugString return the debug string for client connection command
func (c *commandImpl) DebugString() string {
	return fmt.Sprintf("kubectl exec %s -c %s -n %s -- %s",
		c.podFrom, c.containerFrom, c.nsFrom, strings.Join(c.cmd, " "))
}

func (c *commandImpl) Execute(config *rest.Config, cs *kubernetes.Clientset) (stdout, stderr string, err error) {
	return ek.ExecWithOptions(config, cs, &ek.ExecOptions{
		Command:            c.cmd,
		Namespace:          c.nsFrom,
		PodName:            c.podFrom,
		ContainerName:      c.containerFrom,
		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}
