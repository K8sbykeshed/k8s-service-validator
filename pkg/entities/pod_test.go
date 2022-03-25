package entities

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

const (
	sampleIP1 = "10.0.0.1"
	sampleIP2 = "10.0.0.2"
	sampleIP3 = "10.0.0.3"
)

var _ = Describe("external IP test", func() {
	var externalIP ExternalIP
	BeforeEach(func() {
		externalIP = NewExternalIP(sampleIP1, v1.ProtocolTCP)
	})
	It("should create external IP", func() {
		Expect(externalIP.IP).To(Equal(sampleIP1))
		Expect(externalIP.Protocol).To(Equal(v1.ProtocolTCP))
	})
})

var _ = Describe("external IPs array test", func() {
	var ips []string
	var protocol v1.Protocol

	BeforeEach(func() {
		ips = []string{sampleIP1, sampleIP2, sampleIP3}
		protocol = v1.ProtocolTCP
	})

	It("should create array of external IPs", func() {
		externalIPs := NewExternalIPs(ips, protocol)
		Expect(externalIPs).To(HaveLen(3))
		Expect(externalIPs[1].Protocol).To(Equal(v1.ProtocolTCP))
		Expect(externalIPs[1].IP).To(Equal(sampleIP2))
	})
})

var _ = Describe("pod test", func() {
	var pod Pod

	BeforeEach(func() {
		pod = Pod{
			Namespace: "test-ns",
			Name:      "my-pod",
			NodeName:  "my-node",
		}
	})

	Context("toPort", func() {
		BeforeEach(func() {
			pod.ToPort = 8080
		})
		It("can get toPort", func() {
			Expect(pod.GetToPort()).To(Equal(int32(8080)))
		})
		It("can set toPort", func() {
			pod.SetToPort(8081)
			Expect(pod.GetToPort()).To(Equal(int32(8081)))
		})
	})

	Context("hostIP", func() {
		BeforeEach(func() {
			pod.HostIP = sampleIP1
		})
		It("can get toPort", func() {
			Expect(pod.GetHostIP()).To(Equal(sampleIP1))
		})
		It("can set toPort", func() {
			pod.SetHostIP(sampleIP2)
			Expect(pod.GetHostIP()).To(Equal(sampleIP2))
		})
	})

	Context("clusterIP", func() {
		BeforeEach(func() {
			pod.clusterIP = sampleIP1
		})
		It("can get clusterIP", func() {
			Expect(pod.GetClusterIP()).To(Equal(sampleIP1))
		})
		It("can set clusterIP", func() {
			pod.SetClusterIP(sampleIP2)
			Expect(pod.GetClusterIP()).To(Equal(sampleIP2))
		})
	})

	Context("can get external IPs by protocol", func() {
		BeforeEach(func() {
			pod.ExternalIPs = []ExternalIP{
				{
					IP:       sampleIP1,
					Protocol: v1.ProtocolTCP,
				},
				{
					IP:       sampleIP2,
					Protocol: v1.ProtocolUDP,
				},
				{
					IP:       sampleIP3,
					Protocol: v1.ProtocolSCTP,
				},
			}
		})
		It("can get external IPs by certain protocol", func() {
			tcpExternalIPs := pod.GetExternalIPsByProtocol(v1.ProtocolTCP)
			udpExternalIPs := pod.GetExternalIPsByProtocol(v1.ProtocolUDP)
			sctpExternalIPs := pod.GetExternalIPsByProtocol(v1.ProtocolSCTP)
			Expect(tcpExternalIPs).To(HaveLen(1))
			Expect(udpExternalIPs).To(HaveLen(1))
			Expect(sctpExternalIPs).To(HaveLen(1))
			Expect(tcpExternalIPs[0]).To(Equal(NewExternalIP(sampleIP1, v1.ProtocolTCP)))
			Expect(udpExternalIPs[0]).To(Equal(NewExternalIP(sampleIP2, v1.ProtocolUDP)))
			Expect(sctpExternalIPs[0]).To(Equal(NewExternalIP(sampleIP3, v1.ProtocolSCTP)))
		})
	})

	Context("pod string", func() {
		It("should get corresponding pod string", func() {
			podString := pod.PodString()
			Expect(podString.String()).To(Equal("my-node/test-ns/my-pod"))
			Expect(podString.PodName()).To(Equal("my-pod"))
			Expect(podString.Namespace()).To(Equal("test-ns"))
		})
	})

	Context("service name / address", func() {
		It("should returns correct unqualified service name", func() {
			Expect(pod.ServiceName()).To(Equal("s-test-ns-my-pod"))
		})

		It("should return dns domain is provided", func() {
			Expect(pod.QualifiedServiceAddress("example.com")).To(Equal("s-test-ns-my-pod.test-ns.svc.example.com"))
		})
	})

	Context("pod selector", func() {
		It("should returns default labels", func() {
			Expect(pod.LabelSelector()).To(HaveKeyWithValue("pod", "my-pod"))
		})
	})

	Context("convert to k8s pod", func() {
		BeforeEach(func() {
			pod.Containers = []*Container{
				{Name: "nginx", Image: "nginx:1.14", Port: 80},
			}
			pod.HostNetwork = true
		})
		It("can create correct k8s pod object", func() {
			k8sPod := pod.ToK8SSpec()
			Expect(k8sPod.Spec.NodeName).To(Equal("my-node"))
			Expect(k8sPod.Spec.Containers).To(HaveLen(1))
			Expect(*k8sPod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(0)))
			Expect(k8sPod.Spec.HostNetwork).To(BeTrue())
			Expect(k8sPod.Spec.InitContainers).To(BeNil())
			Expect(k8sPod.ObjectMeta.Labels).To(HaveKeyWithValue("pod", "my-pod"))
		})
	})

	Context("reset pod", func() {
		BeforeEach(func() {
			pod.ExternalIPs = NewExternalIPs([]string{sampleIP1, sampleIP2, sampleIP3}, v1.ProtocolTCP)
			pod.SkipProbe = true
			pod.serviceName = "svc"
			pod.clusterIP = sampleIP1
			pod.ToPort = 8080
			pod.HostNetwork = true
		})

		It("should reset fields", func() {
			pod.Reset()
			Expect(pod.ExternalIPs).To(BeNil())
			Expect(pod.SkipProbe).To(BeFalse())
			Expect(pod.serviceName).To(BeEmpty())
			Expect(pod.clusterIP).To(BeEmpty())
			Expect(pod.ToPort).To(BeZero())
			Expect(pod.HostNetwork).To(BeFalse())
		})
	})
})
