package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cronitorio/cronitor-cli/lib"
	"github.com/spf13/viper"
	v1 "k8s.io/api/batch/v1"
)

func (api CronitorApi) mainApiUrl() string {
	if hostnameOverride := viper.GetString("hostname-override"); hostnameOverride != "" {
		return fmt.Sprintf("%s/api", hostnameOverride)
	}
	return "https://cronitor.io/api"
}

func (api CronitorApi) monitorUrl() string {
	// MUST have trailing slash, or will return a 200 with no errors but won't work
	return fmt.Sprintf("%s/monitors", api.mainApiUrl())
}

func (api CronitorApi) PutCronJob(cronJob *v1.CronJob) ([]*lib.Monitor, error) {
	return api.PutCronJobs([]*v1.CronJob{cronJob})
}

func (api CronitorApi) PutCronJobs(cronJobs []*v1.CronJob) ([]*lib.Monitor, error) {
	// Some of this borrowed from https://github.com/cronitorio/cronitor-cli/blob/a5e2b681c89ff8fd5803551206d7ce9674122bd1/lib/cronitor.go
	url := api.monitorUrl()
	if api.IsAutoDiscover {
		url = url + "?auto-discover=1"
	}

	monitorsArray := make([]CronitorJob, 0)
	for _, cronjob := range cronJobs {
		monitorsArray = append(monitorsArray, convertCronJobToCronitorJob(cronjob))
	}

	jsonBytes, err := json.Marshal(monitorsArray)
	if err != nil {
		return nil, err
	}

	slog.Debug("sending request",
		"url", url,
		"body", string(jsonBytes))

	if api.DryRun {
		return make([]*lib.Monitor, 0), nil
	}

	response, err := api.sendHttpPut(url, string(jsonBytes))
	if err != nil {
		return nil, err
	}

	slog.Debug("received response", "response", string(response))

	var responseMonitors []*lib.Monitor
	if err = json.Unmarshal(response, &responseMonitors); err != nil {
		return nil, fmt.Errorf("error from %s: %s, error: %s", url, response, err.Error())
	}

	return responseMonitors, nil
}

func (api CronitorApi) sendHttpRequest(method string, url string, body string) ([]byte, error) {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(api.ApiKey, "")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("User-Agent", api.UserAgent)
	request.Header.Add("Cronitor-Version", "2020-10-27")

	response, err := client.Do(request)
	if err != nil {
		return nil, CronitorApiError{err, response}
	}
	if response.StatusCode != 200 && response.StatusCode != 201 {
		return nil, CronitorApiError{
			fmt.Errorf("error response code %d returned", response.StatusCode),
			response,
		}
	}

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	response.Body = ioutil.NopCloser(bytes.NewBuffer(contents))

	return contents, nil
}

func (api CronitorApi) sendHttpPost(url string, body string) ([]byte, error) {
	return api.sendHttpRequest("POST", url, body)
}

func (api CronitorApi) sendHttpPut(url string, body string) ([]byte, error) {
	return api.sendHttpRequest("PUT", url, body)
}
