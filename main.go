package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/cronitorio/cronitor-kubernetes/cmd"
	"github.com/getsentry/sentry-go"
)

func init() {
	// Configure slog with JSON handler (or TextHandler if you prefer)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	if os.Getenv("SENTRY_ENABLED") == "true" {
		slog.Info("Enabling Sentry instrumentation...")
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              "https://e36895dc862642deae6ba3773924d1f6@o131626.ingest.sentry.io/6031178",
			AttachStacktrace: true,
		})
		if err != nil {
			slog.Error("sentry initialization failed", "error", err)
			os.Exit(1)
		}
		defer sentry.Flush(2 * time.Second)
		defer sentry.Recover()

		if email := os.Getenv("SUPPORT_EMAIL_ADDRESS"); email != "" {
			sentry.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetContext("userEmail", email)
			})
		}
	}

	cmd.Execute()
}
