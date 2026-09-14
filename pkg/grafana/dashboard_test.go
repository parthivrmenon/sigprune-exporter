package grafana

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

func loadDashboard(t *testing.T, path string) DashboardResponse {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	var d DashboardResponse
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatalf("failed to decode fixture: %v", err)
	}
	return d
}

// Both fixtures are the same dashboard fetched from Grafana 13.1: once with its
// rows expanded, and once saved with both rows collapsed, which moves every
// panel into its row's "panels" array. The exprs found must be identical.
func TestGetDashboardPanelExprsCollapsedRows(t *testing.T) {
	expanded := GetDashboardPanelExprs(loadDashboard(t, "testdata/dashboard_rows_expanded.json"))
	collapsed := GetDashboardPanelExprs(loadDashboard(t, "testdata/dashboard_rows_collapsed.json"))

	if len(expanded) != 28 {
		t.Fatalf("expected 28 exprs from the expanded dashboard, got %d", len(expanded))
	}

	sort.Strings(expanded)
	sort.Strings(collapsed)
	if !reflect.DeepEqual(expanded, collapsed) {
		t.Errorf("collapsed rows changed the exprs found: expanded has %d, collapsed has %d", len(expanded), len(collapsed))
	}
}
