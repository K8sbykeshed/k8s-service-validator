package commands

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Netcat command test", func() {
	var client Client

	Context("Netcat client test", func() {
		It("render correct connect command", func() {
			client = NewNcClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolTCP)
			Expect(client.ConnectCommand()).To(Equal([]string{"nc", "-w10", "192.168.0.2", "8080"}))

			client = NewNcClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolUDP)
			Expect(client.ConnectCommand()).To(Equal([]string{"nc", "-u", "-w10", "192.168.0.2", "8080"}))

			client = NewNcClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolSCTP)
			Expect(client.ConnectCommand()).To(BeNil())
		})
	})
})
