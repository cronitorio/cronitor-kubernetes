package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"time"
)

/*
 Sending logs:
Host:  https://logs.cronitor.link/<api key>/<monitor key>/?series=<same series as pings>&metric=length:<byte length before gzip>
Body: <gzipped log message>
*/


func (api CronitorApi) logPresignUrl() string {
	url := fmt.Sprintf("%s/logs/presign", api.mainApiUrl())
	if dev := viper.GetBool("dev"); dev {
		url = url + "?dev=true"
	}
	return url
}

func gzipLogData(logData string) *bytes.Buffer {
	var b bytes.Buffer
	if len(logData) < 1 {
		return &b
	}

	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(logData)); err != nil {
		log.Fatal(errors.Wrap(err, "error writing gzip"))
	}
	if err := gz.Close(); err != nil {
		log.Fatal(errors.Wrap(err, "error closing gzip"))
	}
	return &b
}

func (api CronitorApi) ShipLogData(params *TelemetryEvent) ([]byte, error) {
	cronitorID := pkg.NewCronitorConfigParser(params.CronJob).GetCronitorID()
	seriesID := string(*params.Series)
	gzippedLogs := gzipLogData(params.ErrorLogs)

	jsonBytes, err := json.Marshal(map[string]string{
		"job_key": cronitorID,
		"series": seriesID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't encode job and series IDs to JSON")
	}

	if api.DryRun {
		return nil, nil
	}

	var responseJson struct {
		Url string `json:"url"`
	}
	response, err := api.sendHttpPost(api.logPresignUrl(), string(jsonBytes))
	if err != nil {
		return nil, errors.Wrap(err, "error generating presign url for log uploading")
	}
	if err := json.Unmarshal(response, &responseJson); err != nil {
		return nil, err
	}
	s3LogPutUrl := responseJson.Url
	if len(s3LogPutUrl) == 0 {
		return nil, errors.New("no presigned S3 url returned. Something is wrong")
	}

	req, err := http.NewRequest("PUT", s3LogPutUrl, gzippedLogs)
	// In order to add **any** type of headers, this also needs to be adjusted in the
	//req.Header.Add("Content-Type", "text/plain")
	//req.Header.Add("Content-Encoding", "gzip")
	if err != nil {
		return nil, err
	}

	if api.DryRun {
		return nil, nil
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	response2, err := client.Do(req)
	if err != nil || response == nil {
		return nil, CronitorApiError{
			Err:      err,
			Response: response2,
		}
	}
	if response2.StatusCode < 200 || response2.StatusCode >= 300 {
		return nil, CronitorApiError{
			fmt.Errorf("error response code %d returned", response2.StatusCode),
			response2,
		}
	}
	body, err := ioutil.ReadAll(response2.Body)
	if err != nil {
		return nil, err
	}
	defer response2.Body.Close()
	log.Infof("logs shipped for series %s", *params.Series)
	return body, nil
}
