package matrix

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/k8sbykeshed/k8s-service-validator/pkg/entities"
)

var _ = Describe("iperf command test", func() {
	var model *Model

	Context("can create new model", func() {
		model = NewModel([]string{"ns-1", "ns-2"}, []string{"pod-1", "pod-2"}, []int32{8000, 8001}, []v1.Protocol{v1.ProtocolTCP, v1.ProtocolUDP}, "test.local")
		Expect(model.Namespaces).To(HaveLen(2))
		Expect(model.Namespaces[0].Pods).To(HaveLen(2))
		Expect(model.Namespaces[0].Pods[0].Containers).To(HaveLen(4))

		It("should be able to add a new namespace with iperf pods", func() {
			model.AddIPerfNamespace("ns-iperf", []string{"pod-iperf"}, []int32{8080}, []v1.Protocol{v1.ProtocolTCP})
			Expect(model.Namespaces).To(HaveLen(3))
			iperfNS := model.Namespaces[2]
			Expect(iperfNS.Pods).To(HaveLen(1))
			Expect(iperfNS.Pods[0].Containers[0].Image).To(Equal(entities.AgnhostImage))
			Expect(iperfNS.Pods[0].Containers[0].Command).To(Equal([]string{"/usr/bin/iperf", "--server", "--port", "8080"}))
		})
	})
})
