package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cronitorio/cronitor-cli/lib"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/batch/v1beta1"
	"net/http"
	"strings"
	"time"
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
	responseData, err := ioutil.ReadAll(c.Response.Body)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("response: %s, error: %s", responseData, c.Err.Error())
}

func (c *CronitorApiError) Unwrap() error {
	return c.Err
}

func NewCronitorApi(apikey string, dryRun bool) CronitorApi {
	return CronitorApi{
		DryRun:         dryRun,
		UserAgent:      "cronitor-k8s",
		ApiKey:         apikey,
		IsAutoDiscover: true,
	}
}

func (api CronitorApi) Url() string {
	// MUST have trailing slash, or will return a 200 with no errors but won't work
	return "https://cronitor.io/api/monitors/"
}

func (api CronitorApi) PutCronJob(cronJob *v1beta1.CronJob) ([]*lib.Monitor, error) {
	return api.PutCronJobs([]*v1beta1.CronJob{cronJob})
}

func (api CronitorApi) PutCronJobs(cronJobs []*v1beta1.CronJob) ([]*lib.Monitor, error) {
	// Some of this borrowed from https://github.com/cronitorio/cronitor-cli/blob/a5e2b681c89ff8fd5803551206d7ce9674122bd1/lib/cronitor.go
	url := api.Url()
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

	log.Debugf("request: <%s> %s", url, jsonBytes)

	if api.DryRun {
		return make([]*lib.Monitor, 0), nil
	}

	response, err := api.sendHttpPut(url, string(jsonBytes))
	if err != nil {
		return nil, err
	}

	log.Debugf("response: %s", response)

	var responseMonitors []*lib.Monitor
	// TODO: This must be removed and is a temporary hack to enable continued development
	// to get around the fact that the API returns "rules": {} instead of "rules": [] when a monitor
	// has no rules. This is incompatible with the struct declaration of []lib.Rule and breaks
	// the Unmarshal call.
	response = bytes.ReplaceAll(response, []byte(`"rules":{}`), []byte(`"rules":[]`))
	if err = json.Unmarshal(response, &responseMonitors); err != nil {
		return nil, fmt.Errorf("error from %s: %s, error: %s", url, response, err.Error())
	}

	// Do we actually need to do anything with the response yet?

	return responseMonitors, nil
}

func (api CronitorApi) sendHttpPut(url string, body string) ([]byte, error) {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	request, err := http.NewRequest("PUT", url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(api.ApiKey, "")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("User-Agent", api.UserAgent)
	request.Header.Add("Cronitor-Version", "2020-10-27")

	//log.Debug(formatRequest(request))

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

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return contents, nil
}
