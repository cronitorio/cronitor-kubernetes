package main

import (
	"github.com/cronitorio/cronitor-kubernetes/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
	// Set to true to see line number information
	log.SetReportCaller(false)

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
}

func main() {
	cmd.Execute()
}
