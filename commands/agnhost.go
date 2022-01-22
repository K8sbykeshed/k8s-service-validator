package commands

import (
	"fmt"
	"net"
	"strconv"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// agnCommand represents the server or client agnhost command
type agnCommand struct{ commandImpl }

// ConnectCommand returns the client command for connecting to the server
func (c *agnCommand) ConnectCommand() (cmd []string) {
	switch c.protocol {
	case v1.ProtocolTCP:
		cmd = []string{"/agnhost", "connect", net.JoinHostPort(c.addrTo, c.port), "--timeout=5s", "--protocol=tcp"}
	case v1.ProtocolUDP:
		cmd = []string{"/agnhost", "connect", net.JoinHostPort(c.addrTo, c.port), "--timeout=5s", "--protocol=udp"}
	default:
		zap.L().Error(fmt.Sprintf("protocol %s not supported", c.protocol))
	}
	return cmd
}

// ServeCommand returns server's serve command when binding to a port
func (c *agnCommand) ServeCommand() (cmd []string) {
	switch c.protocol {
	case v1.ProtocolTCP:
		cmd = []string{"/agnhost", "serve-hostname", "--tcp", "--http=false", "--port", c.port}
	case v1.ProtocolUDP:
		cmd = []string{"/agnhost", "serve-hostname", "--udp", "--http=false", "--port", c.port}
	default:
		zap.L().Error(fmt.Sprintf("invalid protocol %v", c.protocol))
	}
	return cmd
}

// NewAgnHostClient returns an instance of AgnHost client command
func NewAgnHostClient(nsFrom, podFrom, containerFrom, addrTo string, port int, protocol v1.Protocol) Client {
	agnHost := &agnCommand{commandImpl{
		nsFrom: nsFrom, podFrom: podFrom, containerFrom: containerFrom,
		addrTo: addrTo, port: strconv.Itoa(port), protocol: protocol,
	}}
	agnHost.cmd = agnHost.ConnectCommand()
	return agnHost
}

// NewAgnHostServer returns an instance of AgnHost server command
func NewAgnHostServer(port int, protocol v1.Protocol) Server {
	agnHost := &agnCommand{commandImpl{port: strconv.Itoa(port), protocol: protocol}}
	agnHost.cmd = agnHost.ServeCommand()
	return agnHost
}
