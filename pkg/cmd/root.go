package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var kubeconfig string
var apiKey string

var RootCmd = &cobra.Command{
	Use: "cronitor-k8s",
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
	_ = RootCmd.MarkPersistentFlagRequired("apikey")
}
