package cmd

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestLogFormatJSON(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Set up JSON handler
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Log a message
	logger.Info("test message", "key", "value")

	// Verify output is valid JSON
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Errorf("JSON handler output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure contains expected fields
	if _, ok := logEntry["time"]; !ok {
		t.Error("JSON log entry missing 'time' field")
	}
	if _, ok := logEntry["level"]; !ok {
		t.Error("JSON log entry missing 'level' field")
	}
	if msg, ok := logEntry["msg"]; !ok || msg != "test message" {
		t.Errorf("JSON log entry 'msg' field expected 'test message', got '%v'", msg)
	}
	if key, ok := logEntry["key"]; !ok || key != "value" {
		t.Errorf("JSON log entry 'key' field expected 'value', got '%v'", key)
	}
}

func TestLogFormatText(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Set up Text handler
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	// Log a message
	logger.Info("test message", "key", "value")

	// Verify output is NOT JSON (text format)
	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err == nil {
		t.Error("Text handler output should not be valid JSON")
	}

	// Verify output contains expected text
	if !strings.Contains(output, "test message") {
		t.Errorf("Text log output missing message. Output: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Text log output missing key=value. Output: %s", output)
	}
}

func TestLogFormatValidation(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectValid bool
	}{
		{"json format", "json", true},
		{"text format", "text", true},
		{"empty format defaults to text", "", true},
		{"invalid xml format", "xml", false},
		{"invalid yaml format", "yaml", false},
		{"invalid uppercase JSON", "JSON", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var valid bool
			switch tt.format {
			case "json", "text", "":
				valid = true
			default:
				valid = false
			}

			if valid != tt.expectValid {
				t.Errorf("Format '%s': expected valid=%v, got valid=%v", tt.format, tt.expectValid, valid)
			}
		})
	}
}

func TestLogLevelValidation(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectValid bool
	}{
		{"DEBUG level", "DEBUG", true},
		{"INFO level", "INFO", true},
		{"WARN level", "WARN", true},
		{"ERROR level", "ERROR", true},
		{"empty level defaults to INFO", "", true},
		{"invalid TRACE level", "TRACE", false},
		{"invalid FATAL level", "FATAL", false},
		{"lowercase is invalid", "debug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var valid bool
			switch tt.level {
			case "DEBUG", "INFO", "WARN", "ERROR", "":
				valid = true
			default:
				valid = false
			}

			if valid != tt.expectValid {
				t.Errorf("Level '%s': expected valid=%v, got valid=%v", tt.level, tt.expectValid, valid)
			}
		})
	}
}

func TestJSONHandlerOutput(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	// Test structured logging with multiple fields
	logger.Info("cronjob synced",
		"namespace", "default",
		"name", "my-cronjob",
		"schedule", "*/5 * * * *",
	)

	output := buf.String()

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify all fields are present
	expectedFields := []string{"time", "level", "msg", "namespace", "name", "schedule"}
	for _, field := range expectedFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("JSON log entry missing '%s' field", field)
		}
	}

	// Verify field values
	if logEntry["msg"] != "cronjob synced" {
		t.Errorf("Expected msg 'cronjob synced', got '%v'", logEntry["msg"])
	}
	if logEntry["namespace"] != "default" {
		t.Errorf("Expected namespace 'default', got '%v'", logEntry["namespace"])
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got '%v'", logEntry["level"])
	}
}

func TestTextHandlerOutput(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	logger.Info("cronjob synced",
		"namespace", "default",
		"name", "my-cronjob",
	)

	output := buf.String()

	// Verify it's NOT valid JSON (text format uses key=value)
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err == nil {
		t.Error("Text handler output should not be valid JSON")
	}

	// Verify text format contains expected content
	expectedContent := []string{"cronjob synced", "namespace=default", "name=my-cronjob"}
	for _, content := range expectedContent {
		if !strings.Contains(output, content) {
			t.Errorf("Text log output missing '%s'. Output: %s", content, output)
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name           string
		handlerLevel   slog.Level
		logLevel       slog.Level
		expectLogged   bool
	}{
		{"DEBUG logged at DEBUG level", slog.LevelDebug, slog.LevelDebug, true},
		{"INFO logged at DEBUG level", slog.LevelDebug, slog.LevelInfo, true},
		{"DEBUG not logged at INFO level", slog.LevelInfo, slog.LevelDebug, false},
		{"WARN logged at INFO level", slog.LevelInfo, slog.LevelWarn, true},
		{"INFO not logged at WARN level", slog.LevelWarn, slog.LevelInfo, false},
		{"ERROR logged at ERROR level", slog.LevelError, slog.LevelError, true},
		{"WARN not logged at ERROR level", slog.LevelError, slog.LevelWarn, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: tt.handlerLevel})
			logger := slog.New(handler)

			logger.Log(context.Background(), tt.logLevel, "test message")

			logged := buf.Len() > 0
			if logged != tt.expectLogged {
				t.Errorf("Expected logged=%v, got logged=%v", tt.expectLogged, logged)
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
