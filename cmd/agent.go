package cmd

import (
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	"github.com/cronitorio/cronitor-kubernetes/pkg/collector"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

var dryRun bool

var agentCmd = &cobra.Command{
	PersistentPreRunE: initializeAgentConfig,
	Use:               "agent",
	Short:             "Run the cronitor-kubernetes agent against a Kubernetes cluster",
	RunE:              agentRun,
}

func checkVersion() {
	currentVersion, err := semver.NewVersion(viper.GetString("version"))
	if err != nil {
		log.Errorf("Error parsing version: %s", err)
		return
	}
	latestVersion := pkg.GetLatestVersion()
	if latestVersion == "" {
		log.Errorf("Couldn't get version: %s", currentVersion)
		return
	}
	latestAvailableVersion, err := semver.NewVersion(latestVersion)
	if err != nil {
		log.Errorf("Error parsing version: %s", err)
		return
	}
	constraint, err := semver.NewConstraint("> " + currentVersion.String())
	if err != nil {
		log.Errorf("Error parsing version constraint: %s", err)
		return
	}
	if constraint.Check(latestAvailableVersion) {
		fmt.Printf(`
*************
A new version of cronitor-kubernetes is available!
You are using: %s
Latest version available with Helm: %s
*************
`, currentVersion.String(), latestAvailableVersion.String())
	}
}

func agentRun(cmd *cobra.Command, args []string) error {
	checkVersion()

	apiKey := viper.GetString("apikey")
	if apiKey == "" {
		return errors.New("a Cronitor api key is required. Provide via --apikey or CRONITOR_API_KEY environmental value")
	}
	cronitorApi := api.NewCronitorApi(apiKey, viper.GetBool("dryrun"))
	kubeconfig := viper.GetString("kubeconfig")
	if kubeconfig == "" {
		log.Info("no kubeconfig provided, defaulting to in-cluster...")
	}
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
	agentCmd.Flags().BoolVar(&dryRun, "dryrun", false, "Dry run, do not actually send updates to Cronitor")

	//// Features
	agentCmd.Flags().Bool("ship-logs", false, "Collect and archive the logs from each CronJob run upon completion or failure")

	RootCmd.AddCommand(agentCmd)
}

func initializeAgentConfig(agentCmd *cobra.Command, args []string) error {
	_ = viper.BindPFlag("dryrun", agentCmd.Flags().Lookup("dryrun"))
	_ = viper.BindEnv("ship-logs", "CRONITOR_AGENT_SHIP_LOGS")
	_ = viper.BindPFlag("ship-logs", agentCmd.Flags().Lookup("ship-logs"))

	// We need to add this because declaring PersistentPreRunE in this command
	// overrides the run coming from Root; it doesn't run both
	if RootCmd.PersistentPreRunE != nil {
		return RootCmd.PersistentPreRunE(agentCmd.Parent(), args)
	}

	return nil
}
