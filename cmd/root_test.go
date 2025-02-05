package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestLogLevelParsing(t *testing.T) {
	levelsToTest := []struct {
		levelString string
		level       slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
	}

	for _, tt := range levelsToTest {
		t.Run(fmt.Sprintf("test '%s' level", tt.levelString), func(t *testing.T) {
			level := new(slog.Level)
			err := level.UnmarshalText([]byte(strings.ToUpper(tt.levelString)))
			if err != nil {
				t.Error(err)
			}
			if *level != tt.level {
				t.Errorf("expected level %v, got %v", tt.level, *level)
			}
		})
	}
}

func TestApiKeyFromDocs(t *testing.T) {
	os.Setenv("CRONITOR_API_KEY", "<api key>")
	buffer := new(bytes.Buffer)
	RootCmd.SetOut(buffer)
	err := RootCmd.Execute()
	// we expect this to fail
	if err == nil {
		t.Error("Gave '<api key>' as key, should have returned an error to the user")
	}
}

func TestInvalidApiKey(t *testing.T) {
	os.Setenv("CRONITOR_API_KEY", "k����]}�x�M�k�w��5{���\u0378")
	buffer := new(bytes.Buffer)
	RootCmd.SetOut(buffer)
	err := RootCmd.Execute()
	// we expect this to fail
	if err == nil {
		t.Error("Invalid API key provided; somehow RootCmd executed anyway")
	}
}
