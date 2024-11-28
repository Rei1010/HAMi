package e2e

import (
	"context"
	"log"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Project-HAMi/HAMi/test/utils"
)

var _ = ginkgo.Describe("[Node] Node E2E Tests", func() {
	var clientSet = utils.GetClientSet()
	var nodeName string
	var namespace = "hami-system"
	var testLabelKey = "gpu"
	var testLabelValue = "on"
	var podName = "hami-device-plugin"

	ginkgo.BeforeEach(func() {
		nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(len(nodes.Items)).To(gomega.BeNumerically(">", 0), "No nodes available for testing")

		nodeName = nodes.Items[0].Name
		log.Printf("Using Node: %s for label tests\n", nodeName)
	})

	ginkgo.It("add a label to a node", func() {
		_, err := utils.AddNodeLabel(clientSet, nodeName, testLabelKey, testLabelValue)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		node, err := clientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(node.Labels[testLabelKey]).To(gomega.Equal(testLabelValue), "Label was not correctly added")
	})

	ginkgo.It("check hami device plugin pod status", func() {
		pods, err := utils.GetPods(clientSet, namespace)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.By("Checking if any Pod contains 'test-pod' in its name")
		found := false
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, podName) {
				found = true
				break
			}
		}

		gomega.Expect(found).To(gomega.BeTrue(), "No Pod with name containing '%s' was found", podName)
	})

	//ginkgo.It("remove a label from a node", func() {
	//	_, err := utils.AddNodeLabel(clientSet, nodeName, testLabelKey, testLabelValue)
	//	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	//
	//	_, err = utils.RemoveNodeLabel(clientSet, nodeName, testLabelKey)
	//	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	//
	//	node, err := clientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	//	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	//	_, exists := node.Labels[testLabelKey]
	//	gomega.Expect(exists).To(gomega.BeFalse(), "Label was not correctly removed")
	//})
})
