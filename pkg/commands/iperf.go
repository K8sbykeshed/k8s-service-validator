package commands

import (
	"fmt"
	"strconv"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// iperfCommand represents the server or client iperf command
type iperfCommand struct{ commandImpl }

// ConnectCommand returns the client command for connecting to the server
func (c *iperfCommand) ConnectCommand() (cmd []string) {
	switch c.protocol {
	case v1.ProtocolTCP:
		cmd = []string{"/usr/bin/iperf", "--client", c.addrTo, "--port", c.port, "-yC"}
	case v1.ProtocolUDP:
		cmd = []string{"/usr/bin/iperf", "--client", c.addrTo, "--port", c.port, "--udp", "-yC"}
	case v1.ProtocolSCTP:
		cmd = []string{"/usr/bin/iperf", "--client", c.addrTo, "--port", c.port, "--sctp", "-yC"}
	default:
		zap.L().Error(fmt.Sprintf("protocol %s not supported", c.protocol))
	}
	return cmd
}

// ServeCommand returns server's serve command when binding to a port
func (c *iperfCommand) ServeCommand() (cmd []string) {
	switch c.protocol {
	case v1.ProtocolTCP:
		cmd = []string{"/usr/bin/iperf", "--server", "--port", c.port}
	case v1.ProtocolUDP:
		cmd = []string{"/usr/bin/iperf", "--server", "--port", c.port, "--udp"}
	case v1.ProtocolSCTP:
		cmd = []string{"/usr/bin/iperf", "--server", "--port", c.port, "--sctp"}
	default:
		zap.L().Error(fmt.Sprintf("protocol %s not supported", c.protocol))
	}
	return cmd
}

// NewIPerfClient returns an instance of iperf client command
func NewIPerfClient(nsFrom, podFrom, containerFrom, addrTo string, port int, protocol v1.Protocol) Client {
	iperf := &iperfCommand{commandImpl{
		nsFrom: nsFrom, podFrom: podFrom, containerFrom: containerFrom,
		addrTo: addrTo, port: strconv.Itoa(port), protocol: protocol,
	}}
	iperf.cmd = iperf.ConnectCommand()
	return iperf
}

// NewIPerfServer returns an instance of iperf server command
func NewIPerfServer(port int, protocol v1.Protocol) Server {
	iperf := &iperfCommand{commandImpl{port: strconv.Itoa(port), protocol: protocol}}
	iperf.cmd = iperf.ServeCommand()
	return iperf
}
