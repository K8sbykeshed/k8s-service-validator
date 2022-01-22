package commands

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("iperf command test", func() {
	var client Client
	var server Server

	Context("iperf client test", func() {
		It("render correct connect command", func() {
			client = NewIPerfClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolTCP)
			Expect(client.ConnectCommand()).To(Equal([]string{"/usr/bin/iperf", "--client", "192.168.0.2", "--port", "8080", "-yC"}))

			client = NewIPerfClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolUDP)
			Expect(client.ConnectCommand()).To(Equal([]string{"/usr/bin/iperf", "--client", "192.168.0.2", "--port", "8080", "--udp", "-yC"}))

			client = NewIPerfClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolSCTP)
			Expect(client.ConnectCommand()).To(Equal([]string{"/usr/bin/iperf", "--client", "192.168.0.2", "--port", "8080", "--sctp", "-yC"}))
		})
	})

	Context("iperf server test", func() {
		It("render correct serve command", func() {
			server = NewIPerfServer(8080, v1.ProtocolTCP)
			Expect(server.ServeCommand()).To(Equal([]string{"/usr/bin/iperf", "--server", "--port", "8080"}))

			server = NewIPerfServer(8080, v1.ProtocolUDP)
			Expect(server.ServeCommand()).To(Equal([]string{"/usr/bin/iperf", "--server", "--port", "8080", "--udp"}))

			server = NewIPerfServer(8080, v1.ProtocolSCTP)
			Expect(server.ServeCommand()).To(Equal([]string{"/usr/bin/iperf", "--server", "--port", "8080", "--sctp"}))
		})
	})
})
