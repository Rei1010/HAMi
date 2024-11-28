package utils

import (
	"flag"
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeConfig string

func init() {
	flag.StringVar(&kubeConfig, "kubeconfig", defaultKubeConfigPath(), "Path to the kubeConfig file")
}

func defaultKubeConfigPath() string {
	configPath := os.Getenv("KUBE_CONF")
	if configPath == "" {
		log.Fatalf("Environment variable KUBE_CONF is not set or empty. Please set it to a valid kubeconfig file path.")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Kubeconfig file does not exist at path: %s", configPath)
	}
	return configPath
}

func DefaultKubeConfigPath() string {
	configPath := os.Getenv("KUBE_CONF")
	if configPath == "" {
		log.Fatalf("Environment variable KUBE_CONF is not set or empty. Please set it to a valid kubeconfig file path.")
	}
	log.Println(configPath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {

		log.Fatalf("lalala Kubeconfig file does not exist at path: %s, error is %s", configPath, err)
	}
	return configPath
}

func GetClientSet() *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Fatalf("Failed to load kubeConfig: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	return clientSet
}

//func ExecCommandInPod(clientSet kubernetes.Interface, namespace, podName, containerName string, command []string) (string, error) {
//	req := clientSet.CoreV1().RESTClient().
//		Post().
//		Resource("pods").
//		Name(podName).
//		Namespace(namespace).
//		SubResource("exec").
//		Param("container", containerName)
//	for _, cmd := range command {
//		req = req.Param("command", cmd)
//	}
//	req = req.Param("stdin", "false").
//		Param("stdout", "true").
//		Param("stderr", "true").
//		Param("tty", "false")
//
//	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
//	if err != nil {
//		return "", fmt.Errorf("failed to initialize executor: %v", err)
//	}
//
//	var stdout, stderr bytes.Buffer
//	err = exec.Stream(remotecommand.StreamOptions{
//		Stdout: &stdout,
//		Stderr: &stderr,
//	})
//	if err != nil {
//		return "", fmt.Errorf("failed to execute command: %v", err)
//	}
//
//	return stdout.String(), nil
//}
