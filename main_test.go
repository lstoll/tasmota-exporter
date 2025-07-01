package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTasmotaCollector_Smoke(t *testing.T) {
	// Create a mock Tasmota server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cm" || r.URL.Query().Get("cmnd") != "status 10" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Return a mock Tasmota status response
		response := TasmotaStatus{
			StatusSNS: StatusSNS{
				Time: "2025-01-15T10:30:00",
				ENERGY: Energy{
					TotalStartTime: "2025-01-15T00:00:00",
					Total:          1.5,
					Yesterday:      0.8,
					Today:          0.7,
					Power:          45.2,
					ApparentPower:  50.0,
					ReactivePower:  20.0,
					Factor:         0.9,
					Voltage:        240.0,
					Current:        0.2,
				},
				ESP32: ESP32{
					Temperature: 42.5,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Create outlets pointing to our mock server
	outlets := []Outlet{
		{Name: "test-outlet", IP: mockServer.Listener.Addr().String()},
	}

	// Create collector
	collector := NewTasmotaCollector(outlets)

	// Test Describe - should have 12 metric descriptions (all our gauges)
	expectedDescs := 12
	descCount := testutil.CollectAndCount(collector, "tasmota_up", "tasmota_on", "tasmota_voltage_volts",
		"tasmota_current_amperes", "tasmota_power_watts", "tasmota_apparent_power_voltamperes",
		"tasmota_reactive_power_voltamperesreactive", "tasmota_power_factor", "tasmota_today_kwh_total",
		"tasmota_yesterday_kwh_total", "tasmota_kwh_total", "tasmota_temperature_celsius")

	if descCount != expectedDescs {
		t.Errorf("expected %d metric descriptions, got %d", expectedDescs, descCount)
	}

	// Test Collect - should have 12 metrics (all our gauges for one outlet)
	expectedMetrics := 12
	metricCount := testutil.CollectAndCount(collector)
	if metricCount != expectedMetrics {
		t.Errorf("expected %d metrics, got %d", expectedMetrics, metricCount)
	}

	// Test specific metric values using CollectAndCompare
	expectedMetricsText := `# HELP tasmota_up Indicates if the tasmota outlet is reachable
# TYPE tasmota_up gauge
tasmota_up{outlet="test-outlet"} 1
# HELP tasmota_power_watts current power of tasmota plug in watts (W)
# TYPE tasmota_power_watts gauge
tasmota_power_watts{outlet="test-outlet"} 45.2
# HELP tasmota_temperature_celsius temperature of the ESP32 chip in celsius
# TYPE tasmota_temperature_celsius gauge
tasmota_temperature_celsius{outlet="test-outlet"} 42.5
`

	if err := testutil.CollectAndCompare(collector, strings.NewReader(expectedMetricsText),
		"tasmota_up", "tasmota_power_watts", "tasmota_temperature_celsius"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestTasmotaCollector_ProbeFailure(t *testing.T) {
	// Create a mock server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	outlets := []Outlet{
		{Name: "failing-outlet", IP: mockServer.Listener.Addr().String()},
	}

	collector := NewTasmotaCollector(outlets)

	// Test Collect - should only have 1 metric (up=0) for failed outlet
	metricCount := testutil.CollectAndCount(collector)
	if metricCount != 1 {
		t.Errorf("expected 1 metric for failed outlet, got %d", metricCount)
	}
}

func TestParseOutlets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Outlet
	}{
		{
			name:  "valid single outlet",
			input: "livingroom:192.168.1.100",
			expected: []Outlet{
				{Name: "livingroom", IP: "192.168.1.100"},
			},
		},
		{
			name:  "valid multiple outlets",
			input: "livingroom:192.168.1.100,bedroom:192.168.1.101",
			expected: []Outlet{
				{Name: "livingroom", IP: "192.168.1.100"},
				{Name: "bedroom", IP: "192.168.1.101"},
			},
		},
		{
			name:  "with whitespace",
			input: " livingroom : 192.168.1.100 , bedroom : 192.168.1.101 ",
			expected: []Outlet{
				{Name: "livingroom", IP: "192.168.1.100"},
				{Name: "bedroom", IP: "192.168.1.101"},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []Outlet{},
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: []Outlet{},
		},
		{
			name:     "missing ip",
			input:    "livingroom:",
			expected: []Outlet{},
		},
		{
			name:     "missing name",
			input:    ":192.168.1.100",
			expected: []Outlet{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOutlets(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d outlets, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i].Name != expected.Name || result[i].IP != expected.IP {
					t.Errorf("outlet %d: expected %+v, got %+v", i, expected, result[i])
				}
			}
		})
	}
}
