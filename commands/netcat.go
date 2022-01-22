package commands

import (
	"fmt"
	"strconv"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// ncCommand represents the client netcat command
type ncCommand struct{ commandImpl }

func (c *ncCommand) ConnectCommand() (cmd []string) {
	switch c.protocol {
	case v1.ProtocolTCP:
		cmd = []string{"nc", "-w10", c.addrTo, c.port}
	case v1.ProtocolUDP:
		cmd = []string{"nc", "-u", "-w10", c.addrTo, c.port}
	default:
		zap.L().Error(fmt.Sprintf("protocol %s not supported", c.protocol))
	}
	return cmd
}

// NewNcClient returns an instance of AgnHost client command
func NewNcClient(nsFrom, podFrom, containerFrom, addrTo string, port int, protocol v1.Protocol) Client {
	nc := &ncCommand{commandImpl{
		nsFrom: nsFrom, podFrom: podFrom, containerFrom: containerFrom,
		addrTo: addrTo, port: strconv.Itoa(port), protocol: protocol,
	}}
	nc.cmd = nc.ConnectCommand()
	return nc
}
