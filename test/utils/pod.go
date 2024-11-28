package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var Pod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "gpu-pod",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:    "container-name",
				Image:   "ubuntu:22.04",
				Command: []string{"bash", "-c", "sleep 86400"},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"nvidia.com/gpu":      resource.MustParse("2"),    // 请求 2 个 vGPU
						"nvidia.com/gpumem":   resource.MustParse("3000"), // 每个 vGPU 内存 3000 MiB
						"nvidia.com/gpucores": resource.MustParse("30"),   // 每个 vGPU 使用 30% GPU 核心
					},
				},
			},
		},
	},
}

func CreatePod(clientSet *kubernetes.Clientset, pod *v1.Pod, namespace string) (*v1.Pod, error) {
	createdPod, err := clientSet.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create Pod %s in namespace %s: %v", pod.Name, namespace, err)
		return nil, err
	}
	return createdPod, nil
}

func DeletePod(clientSet *kubernetes.Clientset, podName, namespace string) error {
	err := clientSet.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Failed to delete Pod %s in namespace %s: %v", podName, namespace, err)
		return err
	}
	return nil
}

func GetPods(clientSet *kubernetes.Clientset, namespace string) (*v1.PodList, error) {
	pods, err := clientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list Pods in namespace %s: %v", namespace, err)
		return nil, err
	}
	return pods, nil
}

func WaitForPodRunning(clientset kubernetes.Interface, namespace, podName string) error {
	// 重试等待间隔和超时时间
	const (
		checkInterval = 2 * time.Second
		timeout       = 3 * time.Minute
	)

	return wait.PollImmediate(checkInterval, timeout, func() (bool, error) {
		// 获取目标 Pod
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get pod %s/%s: %v", namespace, podName, err)
		}

		// 检查 Pod 的状态是否为 Running
		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}

		// 如果 Pod 处于失败状态，直接返回错误
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
			return false, fmt.Errorf("pod %s/%s is in failed or unknown state: %s", namespace, podName, pod.Status.Phase)
		}

		// 继续等待
		return false, nil
	})
}
