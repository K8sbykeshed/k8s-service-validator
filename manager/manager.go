package manager

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeManager
type KubeManager struct {
	clientSet *kubernetes.Clientset
}

// NewKubeManager
func NewKubeManager(cs *kubernetes.Clientset) *KubeManager {
	return &KubeManager{clientSet: cs}
}

// InitializeCluster
func (k *KubeManager) InitializeCluster(model *Model) error {
	var createdPods []*v1.Pod
	for _, ns := range model.Namespaces { // create namespaces
		if _, err := k.CreateNamespace(ns.Spec()); err != nil {
			return err
		}
		for _, pod := range ns.Pods {
			fmt.Println(fmt.Printf("creating/updating pod %s/%s", ns.Name, pod.Name))

			kubePod, err := k.createPod(pod.KubePod())
			if err != nil {
				return err
			}
			createdPods = append(createdPods, kubePod)
		}
	}
	for _, createdPod := range createdPods {
		fmt.Println(createdPod.Status)
		//err := e2epod.WaitForPodRunningInNamespace(k.clientSet, createdPod)
	}
	return nil
}


// createPod is a convenience function for pod setup.
func (k *KubeManager) createPod(pod *v1.Pod) (*v1.Pod, error) {
	ns := pod.Namespace
	fmt.Println(fmt.Printf("creating pod %s/%s", ns, pod.Name))
	createdPod, err := k.clientSet.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update pod %s/%s", ns, pod.Name)
	}
	return createdPod, nil
}

// CreateNamespace
func (k *KubeManager) CreateNamespace(ns *v1.Namespace) (*v1.Namespace, error) {
	createdNamespace, err := k.clientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update namespace %s", ns.Name)
	}
	return createdNamespace, nil
}

// DeleteNamespaces
func (k *KubeManager) DeleteNamespaces(namespaces []string) error {
	for _, ns := range namespaces {
		err := k.clientSet.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to delete namespace %s", ns)
		}
	}
	return nil
}
