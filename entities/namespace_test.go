package entities

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("namespace unit test", func() {
	var namespace *Namespace
	Context("to k8s namespace", func() {
		BeforeEach(func() {
			namespace = &Namespace{Name: "test-ns"}
		})
		It("should render k8s namespace object", func() {
			k8sNamespace := namespace.Spec()
			Expect(k8sNamespace.ObjectMeta.Name).To(Equal("test-ns"))
			Expect(k8sNamespace.ObjectMeta.Labels).To(HaveKeyWithValue("ns", "test-ns"))
		})
	})
})
