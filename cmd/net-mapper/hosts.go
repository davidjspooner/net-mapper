package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var hostsDiscovered = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "hosts_discovered",
	Help: "Number of hosts discovered",
}, []string{"source"})

var hostsForgotten = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "hosts_forgetten",
	Help: "Number of hosts forgotten",
}, []string{"source"})

var hostsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "hosts_active",
	Help: "Number of active hosts",
}, []string{"source"})

var hostsLastChanged = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "hosts_last_changed",
	Help: "Timestamp of last change",
}, []string{"source"})

type Hosts struct {
	source     string
	remembered map[string]time.Time
}

func (h *Hosts) Remember(host ...string) {
	if h.remembered == nil {
		h.remembered = make(map[string]time.Time)
	}
	now := time.Now()
	changed := false
	for _, host := range host {
		_, ok := h.remembered[host]
		if !ok {
			changed = true
			hostsDiscovered.WithLabelValues(h.source).Inc()
		}
		h.remembered[host] = now
	}
	if changed {
		hostsActive.WithLabelValues(h.source).Set(float64(len(h.remembered)))
		hostsLastChanged.WithLabelValues(h.source).Set(float64(now.Unix()))
	}
}

func (h *Hosts) ForgetHostsOlderThan(d time.Duration) {
	now := time.Now()
	changed := false
	for host, lastSeen := range h.remembered {
		if now.Sub(lastSeen) > d {
			delete(h.remembered, host)
			hostsForgotten.WithLabelValues(h.source).Inc()
			changed = true
		}
	}
	if changed {
		hostsActive.WithLabelValues(h.source).Set(float64(len(h.remembered)))
		hostsLastChanged.WithLabelValues(h.source).Set(float64(now.Unix()))
	}
}
