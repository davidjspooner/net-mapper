package main

import (
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/publisher"
	"github.com/davidjspooner/net-mapper/internal/report"
)

type TargetConfig struct {
	Name      string             `yaml:"name"`
	Source    []string           `yaml:"sources"`
	Report    []framework.Config `yaml:"reports"`
	Publisher []framework.Config `yaml:"publishers"`
}

type Target struct {
	name      string
	source    []string
	report    framework.PluginMap[report.Interface]
	publisher framework.PluginMap[publisher.Interface]
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
	err = w.report.LoadAll("report."+config.Name, config.Report)
	if err != nil {
		return nil, fmt.Errorf("target %q, %s", config.Name, err)
	}

	w.publisher = framework.PluginMap[publisher.Interface]{
		Class:   "publisher",
		Factory: publisher.NewPublisher,
		Require: framework.SupportName | framework.RequireKind | framework.RequireReports,
	}
	err = w.publisher.LoadAll("publisher."+config.Name, config.Publisher)
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
	return w, nil
}
