package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var kubeconfig string
var apiKey string

var RootCmd = &cobra.Command{
	PersistentPreRunE: initializeConfig,
	Use:               "cronitor-k8s",
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to a kubeconfig to use")
	RootCmd.PersistentFlags().StringVar(&apiKey, "apikey", "", "Cronitor.io API key")
}

func initializeConfig(cmd *cobra.Command, args []string) error {
	_ = viper.BindEnv("apikey", "CRONITOR_API_KEY")
	_ = viper.BindPFlag("apikey", cmd.Flags().Lookup("apikey"))
	_ = viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig"))
	return nil
}
