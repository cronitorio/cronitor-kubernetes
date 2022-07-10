package cmd

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestLogLevelParsing(t *testing.T) {
	levelsToTest := []string{"trace", "TRACE", "debug", "DEBUG", "info", "INFO", "warn", "WARN", "warning", "error", "ERROR"}
	for _, levelString := range levelsToTest {
		t.Run(fmt.Sprintf("test '%s' level", levelString), func(t *testing.T) {
			_, err := log.ParseLevel(levelString)
			if err != nil {
				t.Error(err)
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
