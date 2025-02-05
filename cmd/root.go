package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	PersistentPreRunE: initializeConfig,
	Use:               "cronitor-kubernetes",
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().String("kubeconfig", "", "path to a kubeconfig to use")
	RootCmd.PersistentFlags().String("apikey", "", "Cronitor.io API key")
	RootCmd.PersistentFlags().String("hostname-override", "", "App hostname to use (mainly for testing)")
	RootCmd.PersistentFlags().String("log-level", "", "Minimum log level to print for the agent (DEBUG, INFO, WARN, ERROR)")
	_ = RootCmd.PersistentFlags().MarkHidden("hostname-override")
	RootCmd.PersistentFlags().Bool("dev", false, "Set the CLI to dev mode (for things like logs, etc.)")
	_ = RootCmd.PersistentFlags().MarkHidden("dev")
}

func initializeConfig(cmd *cobra.Command, args []string) error {
	_ = viper.BindPFlag("kubeconfig", cmd.Flags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("hostname-override", cmd.Flags().Lookup("hostname-override"))
	_ = viper.BindPFlag("dev", cmd.Flags().Lookup("dev"))
	_ = viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))
	_ = viper.BindEnv("version", "APP_VERSION")

	_ = viper.BindEnv("apikey", "CRONITOR_API_KEY")
	_ = viper.BindPFlag("apikey", cmd.Flags().Lookup("apikey"))
	apiKey := viper.GetString("apikey")

	if apiKey == "<api key>" {
		message := "A valid api key is required. You used the string '<api key>' as the api key, which is invalid"
		slog.Error(message)
		return errors.New(message)
	} else if matched, _ := regexp.MatchString(`[\w0-9]+`, apiKey); !matched {
		message := "you have provided an invalid API key. Cronitor API keys are comprised only of number and letter characters"
		slog.Error(message)
		return errors.New(message)
	}

	if logLevel := viper.GetString("log-level"); logLevel != "" {
		var level slog.Level
		switch logLevel {
		case "DEBUG":
			level = slog.LevelDebug
		case "INFO":
			level = slog.LevelInfo
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		default:
			return fmt.Errorf("invalid log level: %s", logLevel)
		}

		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
		slog.SetDefault(slog.New(handler))
	}

	return nil
}
