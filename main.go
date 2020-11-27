package main

import (
	"github.com/jdotjdot/Cronitor-k8s/src/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	cmd.Execute()
}
