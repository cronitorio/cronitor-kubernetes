package main

import "github.com/jdotjdot/Cronitor-k8s/src/collector"

func main() {
	collection := collector.NewCronJobCollection()
	collection.LoadAllExistingCronJobs()
	collection.StartWatching()
}