package cmd

import (
	"github.com/jdotjdot/Cronitor-k8s/src/collector"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the cronitor-k8s agent against a Kubernetes cluster",
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	collection, err := collector.NewCronJobCollection(kubeconfig)
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
	RootCmd.AddCommand(agentCmd)
}
