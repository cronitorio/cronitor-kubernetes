package collector

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetConfig() *rest.Config {
	config, err := rest.InClusterConfig()
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