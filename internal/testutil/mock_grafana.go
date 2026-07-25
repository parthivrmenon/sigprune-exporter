package testutil

import (
	"embed"
	"net/http"
	"net/http/httptest"
)

//go:embed testdata
var testdataFS embed.FS

const (
	MockUser       = "notarealuser"
	MockPassword   = "notarealpassword"
	MockDatasource = "notarealdatasource"
)

var queryResponse = []byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"test-job"},"value":[1234567890,"42"]}]}}`)

func NewMockGrafanaServer() *httptest.Server {
	dashboardsData, _ := testdataFS.ReadFile("testdata/dashboards.json")
	dashboardDetailData, _ := testdataFS.ReadFile("testdata/dashboard_detail.json")
	alertRulesData, _ := testdataFS.ReadFile("testdata/alert_rules.json")
	alertRuleHostDownData, _ := testdataFS.ReadFile("testdata/alert_rule_host_down.json")
	alertRuleHighCPUData, _ := testdataFS.ReadFile("testdata/alert_rule_high_cpu.json")
	alertRuleDiskFillingData, _ := testdataFS.ReadFile("testdata/alert_rule_disk_filling.json")
	alertRuleHighMemoryData, _ := testdataFS.ReadFile("testdata/alert_rule_high_memory.json")
	tsdbStatusData, _ := testdataFS.ReadFile("testdata/tsdb_status.json")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != MockUser || password != MockPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/search":
			if r.URL.Query().Get("type") == "dash-db" {
				w.Write(dashboardsData)
			}
		case "/api/dashboards/uid/nodeExporterBasic":
			w.Write(dashboardDetailData)
		case "/api/v1/provisioning/alert-rules":
			w.Write(alertRulesData)
		case "/api/v1/provisioning/alert-rules/node_exporter_host_down":
			w.Write(alertRuleHostDownData)
		case "/api/v1/provisioning/alert-rules/node_exporter_high_cpu":
			w.Write(alertRuleHighCPUData)
		case "/api/v1/provisioning/alert-rules/node_exporter_disk_filling":
			w.Write(alertRuleDiskFillingData)
		case "/api/v1/provisioning/alert-rules/node_exporter_high_memory":
			w.Write(alertRuleHighMemoryData)
		case "/api/datasources/proxy/uid/" + MockDatasource + "/api/v1/status/tsdb":
			w.Write(tsdbStatusData)
		case "/api/datasources/proxy/uid/" + MockDatasource + "/api/v1/query":
			w.Write(queryResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
