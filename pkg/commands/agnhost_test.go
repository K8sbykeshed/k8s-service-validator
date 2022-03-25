package commands

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Agnhost command test", func() {
	var client Client
	var server Server

	Context("Agnhost client test", func() {
		It("render correct connect command", func() {
			client = NewAgnHostClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolTCP)
			Expect(client.ConnectCommand()).To(Equal([]string{"/agnhost", "connect", "192.168.0.2:8080", "--timeout=5s", "--protocol=tcp"}))

			client = NewAgnHostClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolUDP)
			Expect(client.ConnectCommand()).To(Equal([]string{"/agnhost", "connect", "192.168.0.2:8080", "--timeout=5s", "--protocol=udp"}))

			client = NewAgnHostClient("test-ns", "from-pod", "from-container", "192.168.0.2", 8080, v1.ProtocolSCTP)
			Expect(client.ConnectCommand()).To(BeNil())
		})
	})

	Context("Agnhost server test", func() {
		It("render correct serve command", func() {
			server = NewAgnHostServer(8080, v1.ProtocolTCP)
			Expect(server.ServeCommand()).To(Equal([]string{"/agnhost", "serve-hostname", "--tcp", "--http=false", "--port", "8080"}))

			server = NewAgnHostServer(8080, v1.ProtocolUDP)
			Expect(server.ServeCommand()).To(Equal([]string{"/agnhost", "serve-hostname", "--udp", "--http=false", "--port", "8080"}))

			server = NewAgnHostServer(8080, v1.ProtocolSCTP)
			Expect(server.ServeCommand()).To(BeNil())
		})
	})
})
