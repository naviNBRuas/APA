package agent

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (rt *Runtime) prometheusHandler() http.Handler {
	reg := prometheus.NewRegistry()
	collectors := []prometheus.Collector{
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "apa_uptime_seconds", Help: "Agent uptime in seconds"}),
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "apa_goroutines", Help: "Number of goroutines"}),
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "apa_peers", Help: "Number of connected peers"}),
		prometheus.NewCounter(prometheus.CounterOpts{Name: "apa_audit_entries_total", Help: "Total audit log entries"}),
	}
	for _, c := range collectors {
		reg.MustRegister(c)
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

func (rt *Runtime) monitorBinaryIntegrity(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	check := func() {
		if rt.antiTamper == nil || rt.binaryPath == "" {
			return
		}
		data, err := os.ReadFile(rt.binaryPath)
		if err != nil {
			rt.logger.Warn("Failed to read binary for integrity check", "error", err)
			return
		}
		if ok := rt.antiTamper.VerifyIntegrity(data); !ok {
			rt.logger.Error("Binary integrity check failed", "binary", rt.binaryPath)
			if rt.regenerator != nil {
				if err := rt.regenerator.TriggerRegeneration(ctx); err != nil {
					rt.logger.Error("Failed to trigger regeneration after tamper detection", "error", err)
				}
			}
		}
	}

	check()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
