package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
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

func (c CronitorApiError) ResponseBody() ([]byte, error) {
	contents, err := ioutil.ReadAll(c.Response.Body)
	if err != nil {
		slog.Error("could not read response body", "error", err)
		return []byte{}, err
	}
	defer c.Response.Body.Close()
	c.Response.Body = ioutil.NopCloser(bytes.NewBuffer(contents))
	return contents, nil
}

func (c CronitorApiError) Error() string {
	if c.Response != nil {
		defer c.Response.Body.Close()
		// Sometimes the body is already closed here, so we can't read response data, but we can get the URL we tried
		url := c.Response.Request.URL
		body, err := c.ResponseBody()
		if err != nil {
			return fmt.Sprintf("url: %s, error: %s", url, c.Err.Error())
		} else {
			return fmt.Sprintf("url: %s, body: %s, error: %s", url, body, c.Err.Error())
		}
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
		UserAgent:      fmt.Sprintf("cronitor-kubernetes/%s", os.Getenv("APP_VERSION")),
		ApiKey:         apikey,
		IsAutoDiscover: true,
	}
}
