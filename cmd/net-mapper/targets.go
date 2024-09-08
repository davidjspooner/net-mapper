package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/publisher"
	"github.com/davidjspooner/net-mapper/internal/report"
	"github.com/davidjspooner/net-mapper/internal/source"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var hostsDiscovered = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "hosts_discovered",
	Help: "Number of hosts discovered",
}, []string{"job"})

var hostsForgotten = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "hosts_forgetten",
	Help: "Number of hosts forgotten",
}, []string{"job"})

var hostsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "hosts_active",
	Help: "Number of active hosts",
}, []string{"job"})

var hostsLastChanged = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "hosts_last_changed",
	Help: "Timestamp of last change",
}, []string{"job"})

type TargetConfig struct {
	Name        string             `yaml:"name"`
	Source      []string           `yaml:"sources"`
	Report      []framework.Config `yaml:"reports"`
	Publisher   []framework.Config `yaml:"publishers"`
	ForgetAfter job.DurationString `yaml:"forget_after"`
}

type Target struct {
	name            string
	source          []string
	report          framework.PluginMap[report.Interface]
	publisher       framework.PluginMap[publisher.Interface]
	rememberedHosts map[string]time.Time
	lock            sync.RWMutex
	content         map[string]string
}

func NewTarget(config *TargetConfig) (*Target, error) {
	w := &Target{
		name: config.Name,
	}

	var err error
	w.source = config.Source
	if len(w.source) == 0 {
		return nil, fmt.Errorf("target %q has no sources", config.Name)
	}

	if len(config.Report) == 0 {
		return nil, fmt.Errorf("target %q has no reports", config.Name)
	}
	w.report = framework.PluginMap[report.Interface]{
		Class:   "report",
		Factory: report.NewReportGenerator,
		Require: framework.RequireName | framework.SupportKind | framework.SupportSources,
	}
	err = w.report.LoadAll(config.Report)
	if err != nil {
		return nil, fmt.Errorf("target %q, %s", config.Name, err)
	}

	w.publisher = framework.PluginMap[publisher.Interface]{
		Class:   "publisher",
		Factory: publisher.NewPublisher,
		Require: framework.SupportName | framework.RequireKind | framework.RequireReports,
	}
	err = w.publisher.LoadAll(config.Publisher)
	if err != nil {
		return nil, fmt.Errorf("target %q, %s", config.Name, err)
	}

	err = w.publisher.ForEach(func(name string, t *framework.Plugin[publisher.Interface]) error {
		if len(t.Reports) == 0 {
			return fmt.Errorf("publisher %q has no report", name)
		}
		for _, reportName := range t.Reports {
			if _, err := w.report.Find(reportName); err != nil {
				return fmt.Errorf("publisher %q depends on unknown report %s", name, reportName)
			}
		}
		if len(t.Sources) != 0 {
			return fmt.Errorf("publisher %q should depend on a report not a sources", name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("target %q, %s", config.Name, err)
	}

	w.content = make(map[string]string)

	return w, nil
}

func (w *Target) updateMemory(hosts source.HostList, memoryDuration time.Duration) source.HostList {

	now := time.Now()
	changed := false
	for _, host := range hosts {
		if _, ok := w.rememberedHosts[host]; !ok {
			hostsDiscovered.WithLabelValues(w.name).Inc()
			changed = true
		}
		w.rememberedHosts[host] = now
		log.Printf("discovered %s", host)
	}
	hosts = make(source.HostList, 0, len(w.rememberedHosts))
	for host, lastSeen := range w.rememberedHosts {
		age := now.Sub(lastSeen)
		if age > memoryDuration {
			delete(w.rememberedHosts, host)
			hostsForgotten.WithLabelValues(w.name).Inc()
			log.Printf("forgetting %s, last seen %v", host, lastSeen)
			changed = true
		} else {
			hosts = append(hosts, host)
		}
	}
	hostsActive.WithLabelValues(w.name).Set(float64(len(w.rememberedHosts)))
	if changed {
		hostsLastChanged.WithLabelValues(w.name).Set(float64(now.Unix()))
	}
	return hosts
}

func (w *Target) GenerateReports(ctx context.Context, scannedHosts source.HostList, memoryDuration time.Duration) error {

	w.lock.Lock()
	defer w.lock.Unlock()

	activeHosts := w.updateMemory(scannedHosts, memoryDuration)
	err := w.report.ForEach(func(name string, t *framework.Plugin[report.Interface]) error {
		content, err := t.Impl.Generate(ctx, activeHosts)
		if err != nil {
			return fmt.Errorf("report %q: %v", name, err)
		}
		w.content[name] = content
		return nil
	})
	if err != nil {
		return fmt.Errorf("target %q: %s", w.name, err)
	}
	return nil
}

func (w *Target) ListReports() []string {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.report.Names()
}
