package api

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

// Temporarily borrowed from https://github.com/cronitorio/cronitor-cli/blob/a5e2b681c89ff8fd5803551206d7ce9674122bd1/lib/cronitor.go#L44
type CronitorApi struct {
	DryRun         bool
	ApiKey         string
	IsAutoDiscover bool
	UserAgent      string
}

type CronitorApiError struct {
	Err      error
	Response *http.Response
}

func (c CronitorApiError) Error() string {
	if c.Response != nil {
		responseData, err := ioutil.ReadAll(c.Response.Body)
		defer c.Response.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		return fmt.Sprintf("response: %s, error: %s", responseData, c.Err.Error())
	} else {
		return c.Err.Error()
	}
}

func (c *CronitorApiError) Unwrap() error {
	return c.Err
}

func NewCronitorApi(apikey string, dryRun bool) CronitorApi {
	return CronitorApi{
		DryRun:         dryRun,
		UserAgent:      "cronitor-kubernetes",
		ApiKey:         apikey,
		IsAutoDiscover: true,
	}
}
