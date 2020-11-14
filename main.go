package main

import (
	"github.com/jdotjdot/Cronitor-k8s/src/collector"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	collection := collector.NewCronJobCollection()
	collection.LoadAllExistingCronJobs()
	collection.StartWatchingAll()

	gracefulExit := func() {
		collection.StopWatchingAll()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <- c:
		log.Infof("Received signal %s to exit", sig.String())
		gracefulExit()
	}
	// case <-leaderlost
}