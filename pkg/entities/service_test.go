package entities

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("service unit test", func() {
	var pod Pod
	var containers []*Container

	BeforeEach(func() {
		pod = Pod{
			Namespace: "test-ns",
			Name:      "my-pod",
			NodeName:  "my-node",
		}
		SvcID = nil
		containers = []*Container{
			{Name: "container-1", Protocol: v1.ProtocolTCP, Port: 8080},
			{Name: "container-2", Protocol: v1.ProtocolUDP, Port: 8081},
			{Name: "container-3", Protocol: v1.ProtocolSCTP, Port: 8082},
		}
	})

	It("can create k8s service and increment svc id", func() {
		pod.ServiceName()
		k8sService := NewService(&pod)
		Expect(k8sService.ObjectMeta.Name).To(Equal("s-test-ns-my-pod-1"))
		Expect(k8sService.ObjectMeta.Namespace).To(Equal("test-ns"))
		Expect(k8sService.Spec.Selector).To(HaveKeyWithValue("pod", "my-pod"))
		Expect(SvcID).NotTo(BeNil())
		Expect(SvcID.ID).To(Equal(1))
	})

	Context("return port spec from the service", func() {
		It("can select all protocol", func() {
			servicePorts := portFromContainer(containers, Allprotocols)
			Expect(servicePorts).To(HaveLen(3))
			Expect(servicePorts[0].Name).To(Equal("service-port-tcp-8080"))
			Expect(servicePorts[1].Name).To(Equal("service-port-udp-8081"))
			Expect(servicePorts[2].Name).To(Equal("service-port-sctp-8082"))
		})
		It("can select tcp", func() {
			servicePorts := portFromContainer(containers, v1.ProtocolTCP)
			Expect(servicePorts).To(HaveLen(1))
			Expect(servicePorts[0].Name).To(Equal("service-port-tcp-8080"))
		})
		It("can select udp", func() {
			servicePorts := portFromContainer(containers, v1.ProtocolUDP)
			Expect(servicePorts).To(HaveLen(1))
			Expect(servicePorts[0].Name).To(Equal("service-port-udp-8081"))
		})
		It("can select sctp", func() {
			servicePorts := portFromContainer(containers, v1.ProtocolSCTP)
			Expect(servicePorts).To(HaveLen(1))
			Expect(servicePorts[0].Name).To(Equal("service-port-sctp-8082"))
		})
	})

	Context("create service from pod, with different modes", func() {
		BeforeEach(func() {
			pod.Containers = containers
		})
		It("can create cluster ip service", func() {
			service := pod.ClusterIPService()
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeClusterIP))
			Expect(service.Spec.Ports).To(HaveLen(3))
		})
		It("can create nodeport service", func() {
			service := pod.NodePortService()
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeNodePort))
			Expect(service.Spec.Ports).To(HaveLen(3))
		})
		It("can create load balancer service", func() {
			service := pod.LoadBalancerServiceByProtocol(Allprotocols)
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeLoadBalancer))
			Expect(service.Spec.Ports).To(HaveLen(3))
		})
		It("can create external name service", func() {
			service := pod.ExternalNameService("example.com")
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeExternalName))
			Expect(service.Spec.Ports).To(HaveLen(0))
			Expect(service.Spec.ExternalName).To(Equal("example.com"))
		})
	})
})
