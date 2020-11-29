package main

import (
	"github.com/cronitorio/cronitor-kubernetes/src/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
	// Set to true to see line number information
	log.SetReportCaller(false)
}

func main() {
	cmd.Execute()
}
