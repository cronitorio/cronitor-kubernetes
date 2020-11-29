package cmd

import (
	"github.com/cronitorio/cronitor-kubernetes/src/api"
	"github.com/cronitorio/cronitor-kubernetes/src/collector"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var dryRun bool

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the cronitor-k8s agent against a Kubernetes cluster",
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	cronitorApi := api.NewCronitorApi(apiKey, dryRun)
	collection, err := collector.NewCronJobCollection(kubeconfig, &cronitorApi)
	if err != nil {
		return err
	}
	if err := collection.LoadAllExistingCronJobs(); err != nil {
		return err
	}
	collection.StartWatchingAll()

	gracefulExit := func() {
		collection.StopWatchingAll()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		log.Infof("Received signal %s to exit", sig.String())
		gracefulExit()
	}
	// case <-leaderlost

	return nil
}

func init() {
	agentCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run, do not actually send updates to Cronitor")
	RootCmd.AddCommand(agentCmd)
}
