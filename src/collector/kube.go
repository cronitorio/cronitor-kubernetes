package collector

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetConfig() *rest.Config {
	if config, err := rest.InClusterConfig(); err == nil {
		return config
	}
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/JJ/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	return config
}

func GetClientSet() *kubernetes.Clientset {
	config := GetConfig()
	clientset := kubernetes.NewForConfigOrDie(config)
	return clientset
}