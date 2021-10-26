package main

import (
	"github.com/cronitorio/cronitor-kubernetes/cmd"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func init() {
	//log.SetLevel(log.DebugLevel)
	// Set to true to see line number information
	//log.SetReportCaller(false)

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
}

func main() {
	if os.Getenv("SENTRY_ENABLED") == "true" {
		log.Info("Enabling Sentry instrumentation...")
		err := sentry.Init(sentry.ClientOptions{
			Dsn: "https://e36895dc862642deae6ba3773924d1f6@o131626.ingest.sentry.io/6031178",
			AttachStacktrace: true,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
		defer sentry.Recover()
	}

	cmd.Execute()
}
