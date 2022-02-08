package pkg

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
)

const chartRepositoryUrl = "https://cronitorio.github.io/cronitor-kubernetes/index.yaml"

type chartEntry struct {
	ApiVersion  string   `yaml:"ApiVersion"`
	AppVersion  string   `yaml:"AppVersion"`
	Created     string   `yaml:"created"`
	Description string   `yaml:"description"`
	Digest      string   `yaml:"digest"`
	Name        string   `yaml:"name"`
	Urls        []string `yaml:"urls"`
	Version     string   `yaml:"version"`
}

type chart struct {
	ApiVersion string                  `yaml:"ApiVersion"`
	Generated  string                  `yaml:"generated"`
	Entries    map[string][]chartEntry `yaml:"Entries"`
}

func getChartYaml() ([]byte, error) {
	resp, err := http.Get(chartRepositoryUrl)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalln(err)
	}
	return body, nil
}

func extractVersionsFromChart(chartBody []byte) ([]string, error) {
	var data chart
	if err := yaml.Unmarshal(chartBody, &data); err != nil {
		return nil, err
	}

	var versions []string
	for _, row := range data.Entries["cronitor-kubernetes"] {
		versions = append(versions, row.AppVersion)
	}
	return versions, nil
}

func extractLatestVersionFromList(versions []string) (string, error) {
	vs := make([]*semver.Version, len(versions))
	for i, r := range versions {
		v, err := semver.NewVersion(r)
		if err != nil {
			log.Errorf("Error parsing version: %s", err)
		}

		vs[i] = v
	}
	sort.Sort(semver.Collection(vs))
	if len(vs) <= 1 {
		return "", fmt.Errorf("no versions found: %v", vs)
	}
	lastItem := vs[len(vs)-1]
	return lastItem.String(), nil
}

func GetLatestVersion() string {
	data, err := getChartYaml()
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	versions, err := extractVersionsFromChart(data)
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	latestVersion, err := extractLatestVersionFromList(versions)
	if err != nil {
		return ""
	}
	return strings.Trim(latestVersion, "v")
}
