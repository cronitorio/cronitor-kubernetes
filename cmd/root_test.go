package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
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
