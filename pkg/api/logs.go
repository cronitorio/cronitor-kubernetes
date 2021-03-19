package api

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

/*
 Sending logs:
Host:  https://logs.cronitor.link/<api key>/<monitor key>/?series=<same series as pings>&metric=length:<byte length before gzip>
Body: <gzipped log message>
*/

func (api CronitorApi) logUrl(params *TelemetryEvent) string {
	cronitorID := pkg.NewCronitorConfigParser(params.CronJob).GetCronitorID()
	return fmt.Sprintf("https://logs.cronitor.link/%s/%s/", api.ApiKey, cronitorID)
}

func (t *TelemetryEvent) EncodeForLogs() string {
	q := url.Values{}
	if t.ErrorLogs != "" {
		byteLength := len(t.ErrorLogs)
		q.Add("metric", fmt.Sprintf("length:%d", byteLength))
	}
	if t.Series != nil {
		q.Add("series", string(*t.Series))
	}
	return q.Encode()
}

func gzipLogData(logData string) *bytes.Buffer {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(logData)); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	return &b
}

func (api CronitorApi) ShipLogData(params *TelemetryEvent) ([]byte, error) {
	logUrl := api.logUrl(params)
	gzippedLogs := gzipLogData(params.ErrorLogs)
	req, err := http.NewRequest("POST", logUrl, gzippedLogs)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.EncodeForLogs()

	if api.DryRun {
		return nil, nil
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, CronitorApiError{
			Err:      err,
			Response: response,
		}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, CronitorApiError{
			fmt.Errorf("error response code %d returned", response.StatusCode),
			response,
		}
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
