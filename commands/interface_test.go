package commands

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
)

var _ = Describe("commandImpl test", func() {
	Context("get debug string", func() {
		impl := commandImpl{
			nsFrom:        "from-ns",
			podFrom:       "from-pod",
			containerFrom: "from-container",
			addrTo:        "192.168.0.2",
			port:          "8080",
			protocol:      v1.ProtocolTCP,
		}
		impl.cmd = []string{"cmd", "arg1", "arg2"}

		It("prints correct debug string", func() {
			Expect(impl.DebugString()).To(Equal("kubectl exec from-pod -c from-container -n from-ns -- cmd arg1 arg2"))
		})
	})
})
