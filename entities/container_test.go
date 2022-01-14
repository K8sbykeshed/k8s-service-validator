package entities

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("container unit test", func() {
	var container *Container
	Context("container name", func() {
		BeforeEach(func() {
			container = &Container{Name: "test-container", Protocol: v1.ProtocolTCP}
		})

		It("returns the correct name if provided", func() {
			Expect(container.GetName()).To(Equal("test-container"))
		})

		It("returns hash if port is not provided", func() {
			container = &Container{Name: "", Protocol: v1.ProtocolTCP}
			Expect(container.GetName()).To(HavePrefix("cont-"))
			Expect(container.GetName()).To(Not(HaveSuffix("-tcp")))
		})

		It("returns port and protocol if provided", func() {
			container = &Container{Name: "", Protocol: v1.ProtocolTCP, Port: 8080}
			Expect(container.GetName()).To(Equal("cont-8080-tcp"))
		})
	})

	Context("port name", func() {
		When("port is 0", func() {
			BeforeEach(func() {
				container = &Container{Port: 0}
			})
			It("returns serve appended by 5 digit hash", func() {
				portName := container.PortName()
				Expect(portName).To(HavePrefix("serve-"))
				port, err := strconv.Atoi(portName[6 : len(portName)-1])
				Expect(err).To(BeNil())
				Expect(port < 1e5).To(BeTrue())
			})
		})
		When("port is not zero", func() {
			BeforeEach(func() {
				container = &Container{Port: 8080, Protocol: v1.ProtocolTCP}
			})
			It("returns serve appended by port and protocol", func() {
				Expect(container.PortName()).To(Equal("serve-8080-tcp"))
			})
		})
	})

	Context("to k8s spec", func() {
		BeforeEach(func() {
			container = &Container{Port: 8080, Protocol: v1.ProtocolTCP}
		})
		It("should render correct k8s spec", func() {
			k8sContainer := container.ToK8SSpec()
			Expect(k8sContainer.Name).To(Equal("cont-8080-tcp"))
			Expect(k8sContainer.ImagePullPolicy).To(Equal(v1.PullIfNotPresent))
			Expect(k8sContainer.Image).To(Equal("k8s.gcr.io/e2e-test-images/agnhost:2.31"))
			Expect(k8sContainer.Command).To(Equal([]string{"/agnhost", "serve-hostname", "--tcp", "--http=false", "--port", "8080"}))
			Expect(k8sContainer.Ports[0].Name).To(Equal("serve-8080-tcp"))
		})
	})
})
