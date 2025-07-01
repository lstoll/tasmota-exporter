package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/term"
)

// Outlet represents a tasmota outlet with its name and IP
type Outlet struct {
	Name string
	IP   string
}

// TasmotaCollector implements the prometheus.Collector interface
type TasmotaCollector struct {
	outlets []Outlet

	// Metrics
	onGauge            *prometheus.GaugeVec
	voltageGauge       *prometheus.GaugeVec
	currentGauge       *prometheus.GaugeVec
	powerGauge         *prometheus.GaugeVec
	apparentPowerGauge *prometheus.GaugeVec
	reactivePowerGauge *prometheus.GaugeVec
	factorGauge        *prometheus.GaugeVec
	todayGauge         *prometheus.GaugeVec
	yesterdayGauge     *prometheus.GaugeVec
	totalGauge         *prometheus.GaugeVec
	upGauge            *prometheus.GaugeVec
	temperatureGauge   *prometheus.GaugeVec
}

// NewTasmotaCollector creates a new collector for the given outlets
func NewTasmotaCollector(outlets []Outlet) *TasmotaCollector {
	return &TasmotaCollector{
		outlets: outlets,
		onGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_on",
			Help: "Indicates if the tasmota plug is on/off",
		}, []string{"outlet"}),
		voltageGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_voltage_volts",
			Help: "voltage of tasmota plug in volt (V)",
		}, []string{"outlet"}),
		currentGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_current_amperes",
			Help: "current of tasmota plug in ampere (A)",
		}, []string{"outlet"}),
		powerGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_power_watts",
			Help: "current power of tasmota plug in watts (W)",
		}, []string{"outlet"}),
		apparentPowerGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_apparent_power_voltamperes",
			Help: "apparent power of tasmota plug in volt-amperes (VA)",
		}, []string{"outlet"}),
		reactivePowerGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_reactive_power_voltamperesreactive",
			Help: "reactive power of tasmota plug in volt-amperes reactive (VAr)",
		}, []string{"outlet"}),
		factorGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_power_factor",
			Help: "power factor of tasmota plug",
		}, []string{"outlet"}),
		todayGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_today_kwh_total",
			Help: "todays energy usage total in kilowatts hours (kWh)",
		}, []string{"outlet"}),
		yesterdayGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_yesterday_kwh_total",
			Help: "yesterdays energy usage total in kilowatts hours (kWh)",
		}, []string{"outlet"}),
		totalGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_kwh_total",
			Help: "total energy usage in kilowatts hours (kWh)",
		}, []string{"outlet"}),
		upGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_up",
			Help: "Indicates if the tasmota outlet is reachable",
		}, []string{"outlet"}),
		temperatureGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "tasmota_temperature_celsius",
			Help: "temperature of the ESP32 chip in celsius",
		}, []string{"outlet"}),
	}
}

// Describe implements prometheus.Collector
func (c *TasmotaCollector) Describe(ch chan<- *prometheus.Desc) {
	c.onGauge.Describe(ch)
	c.voltageGauge.Describe(ch)
	c.currentGauge.Describe(ch)
	c.powerGauge.Describe(ch)
	c.apparentPowerGauge.Describe(ch)
	c.reactivePowerGauge.Describe(ch)
	c.factorGauge.Describe(ch)
	c.todayGauge.Describe(ch)
	c.yesterdayGauge.Describe(ch)
	c.totalGauge.Describe(ch)
	c.upGauge.Describe(ch)
	c.temperatureGauge.Describe(ch)
}

// Collect implements prometheus.Collector
func (c *TasmotaCollector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	// Collect metrics for each outlet in parallel
	for _, outlet := range c.outlets {
		wg.Add(1)
		go func(outlet Outlet) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			status, err := c.probeTasmota(ctx, outlet.IP)

			if err != nil {
				slog.Warn("outlet probe failed", "outlet", outlet.Name, "ip", outlet.IP, "error", err)
				// Only send up=0 metric for failed outlets, omit all other metrics
				upMetric := c.upGauge.WithLabelValues(outlet.Name)
				upMetric.Set(0)
				upMetric.Collect(ch)
				return
			}

			slog.Info("outlet probe successful", "outlet", outlet.Name, "ip", outlet.IP)

			// Send up metric immediately
			upMetric := c.upGauge.WithLabelValues(outlet.Name)
			upMetric.Set(1)
			upMetric.Collect(ch)

			// Send all other metrics for this outlet
			if status.StatusSNS.ENERGY.Power > 0 {
				onMetric := c.onGauge.WithLabelValues(outlet.Name)
				onMetric.Set(1)
				onMetric.Collect(ch)
			} else {
				onMetric := c.onGauge.WithLabelValues(outlet.Name)
				onMetric.Set(0)
				onMetric.Collect(ch)
			}

			voltageMetric := c.voltageGauge.WithLabelValues(outlet.Name)
			voltageMetric.Set(status.StatusSNS.ENERGY.Voltage)
			voltageMetric.Collect(ch)

			currentMetric := c.currentGauge.WithLabelValues(outlet.Name)
			currentMetric.Set(status.StatusSNS.ENERGY.Current)
			currentMetric.Collect(ch)

			powerMetric := c.powerGauge.WithLabelValues(outlet.Name)
			powerMetric.Set(status.StatusSNS.ENERGY.Power)
			powerMetric.Collect(ch)

			apparentPowerMetric := c.apparentPowerGauge.WithLabelValues(outlet.Name)
			apparentPowerMetric.Set(status.StatusSNS.ENERGY.ApparentPower)
			apparentPowerMetric.Collect(ch)

			reactivePowerMetric := c.reactivePowerGauge.WithLabelValues(outlet.Name)
			reactivePowerMetric.Set(status.StatusSNS.ENERGY.ReactivePower)
			reactivePowerMetric.Collect(ch)

			factorMetric := c.factorGauge.WithLabelValues(outlet.Name)
			factorMetric.Set(status.StatusSNS.ENERGY.Factor)
			factorMetric.Collect(ch)

			todayMetric := c.todayGauge.WithLabelValues(outlet.Name)
			todayMetric.Set(status.StatusSNS.ENERGY.Today)
			todayMetric.Collect(ch)

			yesterdayMetric := c.yesterdayGauge.WithLabelValues(outlet.Name)
			yesterdayMetric.Set(status.StatusSNS.ENERGY.Yesterday)
			yesterdayMetric.Collect(ch)

			totalMetric := c.totalGauge.WithLabelValues(outlet.Name)
			totalMetric.Set(status.StatusSNS.ENERGY.Total)
			totalMetric.Collect(ch)

			temperatureMetric := c.temperatureGauge.WithLabelValues(outlet.Name)
			temperatureMetric.Set(status.StatusSNS.ESP32.Temperature)
			temperatureMetric.Collect(ch)
		}(outlet)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

func (c *TasmotaCollector) probeTasmota(ctx context.Context, target string) (*TasmotaStatus, error) {
	// Use the JSON API endpoint
	url := fmt.Sprintf("http://%s/cm?cmnd=status%%2010", target)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", target, err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasmota target %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from tasmota target %s: %w", target, err)
	}

	slog.Debug("tasmota target response", "target", target, "response", string(body))

	// Parse JSON response
	var status TasmotaStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response from %s: %w", target, err)
	}

	return &status, nil
}

func main() {
	var (
		listenAddr = flag.String("listen-addr", ":8092", "address to listen on")
		outlets    = flag.String("outlets", "", "comma-separated list of outlet configurations in format 'name:ip' (e.g., 'livingroom:192.168.1.100,bedroom:192.168.1.101')")
		logLevel   = flag.String("log-level", "info", "log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Set up logging
	level := slog.LevelInfo
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		slog.Error("invalid log level", "level", *logLevel, "valid_levels", []string{"debug", "info", "warn", "error"})
		os.Exit(1)
	}

	// Detect if we're running in a TTY (interactive terminal) or not (container/k8s)
	var handler slog.Handler
	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Running in a TTY - use human-readable text format
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		// Running in a container/k8s - use JSON format for better log aggregation
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	if *outlets == "" {
		slog.Error("--outlets flag is required")
		os.Exit(1)
	}

	outletList := parseOutlets(*outlets)
	if len(outletList) == 0 {
		slog.Error("no valid outlet configurations found")
		os.Exit(1)
	}

	slog.Info("configured outlets", "outlets", outletList)

	// Create and register collector
	collector := NewTasmotaCollector(outletList)
	prometheus.MustRegister(collector)

	// Set up metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	slog.Info("starting tasmota exporter", "listen_addr", *listenAddr)
	err := http.ListenAndServe(*listenAddr, nil)
	if errors.Is(err, http.ErrServerClosed) {
		slog.Info("server closed")
	} else if err != nil {
		slog.Error("error starting server", "error", err)
		os.Exit(1)
	}
}

func parseOutlets(outletsStr string) []Outlet {
	var outlets []Outlet

	for _, outletStr := range strings.Split(outletsStr, ",") {
		outletStr = strings.TrimSpace(outletStr)
		if outletStr == "" {
			continue
		}

		parts := strings.Split(outletStr, ":")
		if len(parts) != 2 {
			slog.Warn("invalid outlet configuration", "config", outletStr, "expected_format", "name:ip")
			continue
		}

		name := strings.TrimSpace(parts[0])
		ip := strings.TrimSpace(parts[1])

		if name == "" || ip == "" {
			slog.Warn("invalid outlet configuration", "config", outletStr, "reason", "name and ip cannot be empty")
			continue
		}

		outlets = append(outlets, Outlet{Name: name, IP: ip})
	}

	return outlets
}

// TasmotaStatus represents the JSON response from Tasmota status command
type TasmotaStatus struct {
	StatusSNS StatusSNS `json:"StatusSNS"`
}

// StatusSNS contains the sensor data
type StatusSNS struct {
	Time   string `json:"Time"`
	ENERGY Energy `json:"ENERGY"`
	ESP32  ESP32  `json:"ESP32"`
}

// Energy contains the power monitoring data
type Energy struct {
	TotalStartTime string  `json:"TotalStartTime"`
	Total          float64 `json:"Total"`
	Yesterday      float64 `json:"Yesterday"`
	Today          float64 `json:"Today"`
	Power          float64 `json:"Power"`
	ApparentPower  float64 `json:"ApparentPower"`
	ReactivePower  float64 `json:"ReactivePower"`
	Factor         float64 `json:"Factor"`
	Voltage        float64 `json:"Voltage"`
	Current        float64 `json:"Current"`
}

// ESP32 contains ESP32-specific data like temperature
type ESP32 struct {
	Temperature float64 `json:"Temperature"`
}
