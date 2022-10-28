package collector

import (
	"fmt"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetConfig(pathToKubeconfig string) (*rest.Config, error) {
	// TODO: BuildConfigFromFlags falls back to in-cluster config, so we can remove the later code
	// as long as we put better error messaging in place
	if pathToKubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func GetClientSet(config *rest.Config) *kubernetes.Clientset {
	clientset := kubernetes.NewForConfigOrDie(config)
	return clientset
}

func GetDiscoveryClient(config *rest.Config) *discovery.DiscoveryClient {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Printf(" error in discoveryClient %v", err)
	}
	return discoveryClient
}
