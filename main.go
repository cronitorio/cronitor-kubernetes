package main

import (
	"github.com/jdotjdot/Cronitor-k8s/src/collector"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	collection := collector.NewCronJobCollection()
	collection.LoadAllExistingCronJobs()
	collection.StartWatching()

	select{}
}