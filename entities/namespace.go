package entities

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Namespace defines the structure of namespace
type Namespace struct {
	Name string
	Pods []*Pod
}

// Spec constructs k8s namespace
func (ns *Namespace) Spec() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ns.Name,
			Labels: ns.LabelSelector(),
		},
	}
}

// LabelSelector returns label selector fot the namespace
func (ns *Namespace) LabelSelector() map[string]string {
	return map[string]string{"ns": ns.Name}
}
