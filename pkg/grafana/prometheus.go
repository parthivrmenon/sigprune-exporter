package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	tsdbURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/status/tsdb?limit=%d", datasourceUID, limit)
	req, err := http.NewRequest("GET", c.URL+tsdbURI, nil)
	if err != nil {
		return TSDBStatus{}, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TSDBStatus{}, err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return TSDBStatus{}, err
	}

	var tsdbStatus TSDBStatus
	if err := json.Unmarshal(result, &tsdbStatus); err != nil {
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

func (c *Client) GetPrometheusJobsForMetric(datasourceUID string, metricName string) map[string]int64 {
	queryURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query", datasourceUID)
	req, err := http.NewRequest("GET", c.URL+queryURI, nil)
	if err != nil {
		log.Fatal(err)
	}
	c.setAuth(req)
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

func (c *Client) GetPrometheusJobsForLabel(datasourceUID string, labelName string) map[string]int64 {
	queryURI := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query", datasourceUID)
	req, err := http.NewRequest("GET", c.URL+queryURI, nil)
	if err != nil {
		log.Fatal(err)
	}
	c.setAuth(req)
	params := req.URL.Query()
	params.Add("query", fmt.Sprintf("count by (job) ({%s!=\"\"})", labelName))
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
