package grafana

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type TSDBStatus struct {
	Status   string   `json:"status"`
	TSDBData TSDBData `json:"data"`
}

type TSDBData struct {
	MetricCounts []MetricCount `json:"seriesCountByMetricName"`
	LabelCounts  []LabelCount  `json:"labelValueCountByLabelName"`
}
type MetricCount struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type LabelCount struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

func (c *Client) GetTSDBStatus(datasourceUID string, limit int) (TSDBStatus, error) {
	tsdbURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/status/tsdb", datasourceUID)
	params := url.Values{
		"limit": {strconv.Itoa(limit)},
	}
	var tsdbStatus TSDBStatus
	err := c.getJSON(tsdbURI, params, &tsdbStatus)
	if err != nil {
		return TSDBStatus{}, err
	}

	return tsdbStatus, nil

}

type QueryResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (c *Client) GetPrometheusJobsForMetrics(datasourceUID string, metrics []string) (map[string]map[string]int64, error) {
	// An empty list would build {__name__=~""}, which Prometheus rejects with a 400.
	if len(metrics) == 0 {
		return map[string]map[string]int64{}, nil
	}
	queryURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query", datasourceUID)
	metricNames := strings.Join(metrics, "|")
	query := fmt.Sprintf("count by (job, __name__) ({__name__=~\"%s\"})", metricNames)
	params := url.Values{
		"query": {query},
	}
	var queryResponse QueryResponse
	err := c.getJSON(queryURI, params, &queryResponse)

	if err != nil {
		return nil, err
	}
	metricJobMap := make(map[string]map[string]int64)

	for _, r := range queryResponse.Data.Result {
		metric := r.Metric["__name__"]
		job := r.Metric["job"]
		countStr := r.Value[1].(string) // value[1] is the count as a string
		count, _ := strconv.ParseInt(countStr, 10, 64)

		if metricJobMap[metric] == nil {
			metricJobMap[metric] = make(map[string]int64)
		}
		metricJobMap[metric][job] = count
	}
	return metricJobMap, nil

}

func (c *Client) GetPrometheusJobsForLabel(datasourceUID string, labelName string) (map[string]int64, error) {
	queryURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query", datasourceUID)
	query := fmt.Sprintf("count by (job) ({%s!=\"\"})", labelName)
	params := url.Values{
		"query": {query},
	}

	var queryResponse QueryResponse
	err := c.getJSON(queryURI, params, &queryResponse)
	if err != nil {
		return nil, err
	}
	jobs := make(map[string]int64)
	for _, r := range queryResponse.Data.Result {
		job := r.Metric["job"]
		countStr := r.Value[1].(string) // value[1] is the count as a string
		count, _ := strconv.ParseInt(countStr, 10, 64)
		jobs[job] = count
	}
	return jobs, nil

}
