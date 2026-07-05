package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type MetricCount struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

func (c *Client) GetTopTSDBMetrics(datasourceUID string, limit int) []MetricCount {
	tsdbURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/status/tsdb?limit=%d", datasourceUID, limit)
	req, err := http.NewRequest("GET", c.URL+tsdbURI, nil)
	req.SetBasicAuth(c.Username, c.Password)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var seriesCountByMetricName struct {
		Data struct {
			SeriesCountByMetricName []MetricCount `json:"seriesCountByMetricName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result, &seriesCountByMetricName); err != nil {
		log.Fatal(err)
	}

	return seriesCountByMetricName.Data.SeriesCountByMetricName
}

type QueryResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (c *Client) GetPrometheusJobsForMetric(datasourceUID string, metricName string) map[string]int64 {
	queryURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query", datasourceUID)
	req, err := http.NewRequest("GET", c.URL+queryURI, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(c.Username, c.Password)
	params := req.URL.Query()
	params.Add("query", fmt.Sprintf("count by (job) (%s)", metricName))
	req.URL.RawQuery = params.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var queryResponse QueryResponse
	if err := json.Unmarshal(result, &queryResponse); err != nil {
		log.Fatal(err)
	}
	jobs := make(map[string]int64)
	for _, r := range queryResponse.Data.Result {
		job := r.Metric["job"]
		countStr := r.Value[1].(string) // value[1] is the count as a string
		count, _ := strconv.ParseInt(countStr, 10, 64)
		jobs[job] = count
	}
	return jobs

}
