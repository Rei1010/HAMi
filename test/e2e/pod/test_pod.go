package e2e

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Project-HAMi/HAMi/test/utils"
)

var _ = ginkgo.Describe("Pod E2E Tests", func() {
	var clientSet = utils.GetClientSet()
	var namespace = "default"
	var podName = "e2e-test-pod"

	ginkgo.It("should create a Pod", func() {
		newPod := utils.Pod.DeepCopy()

		createdPod, err := utils.CreatePod(clientSet, newPod, namespace)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(createdPod.Name).To(gomega.Equal(podName), "Pod was not created successfully")

		err = utils.WaitForPodRunning(clientSet, namespace, podName)
		if err != nil {
			t.Fatalf("Failed to wait for GPU Pod running: %v", err)
		}

		//output, err := utils.ExecCommandInPod(clientset, namespace, podName, "container-name", []string{"nvidia-smi"})
		//if err != nil {
		//	t.Fatalf("Failed to execute nvidia-smi: %v", err)
		//}
		//
		//if !strings.Contains(output, "3000 MiB") {
		//	t.Fatalf("Expected 3000 MiB GPU memory in nvidia-smi output, but got: %s", output)
		//}
	})

	ginkgo.It("should delete a Pod", func() {
		err := utils.DeletePod(clientSet, podName, namespace)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		_, err = clientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred(), "Pod was not deleted successfully")
	})
})
