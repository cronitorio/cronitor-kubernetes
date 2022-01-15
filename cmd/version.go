package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Check agent version",
	Run:   versionRun,
}

func versionRun(cmd *cobra.Command, args []string) {
	fmt.Println(viper.GetString("version"))
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
