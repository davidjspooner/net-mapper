package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/source"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ScanFrequency  string             `yaml:"scan_frequency"`
	MemoryDuration string             `yaml:"memory_duration"`
	Sources        []framework.Config `yaml:"sources"`
	Targets        []*TargetConfig    `yaml:"targets"`
}

type Manager struct {
	scanFrequency  time.Duration
	memoryDuration time.Duration

	sources framework.PluginMap[source.Source]
	targets map[string]*Target
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) ReloadConfig(ctx context.Context, configPath string) error {

	config := Config{}

	f, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	err = d.Decode(&config)
	if err != nil {
		return err
	}

	//change directory to config files so other file opens are relative to it
	err = os.Chdir(filepath.Dir(configPath))
	if err != nil {
		return err
	}

	if config.ScanFrequency == "" {
		config.ScanFrequency = "30m" // 100 days
	}
	scanFrequency, err := time.ParseDuration(config.ScanFrequency)
	if err != nil {
		return fmt.Errorf("could not parse scan frequency: %s", err)
	}

	if config.MemoryDuration == "" {
		config.MemoryDuration = "2400h" // 100 days
	}
	memoryDuration, err := time.ParseDuration(config.MemoryDuration)
	if err != nil {
		return fmt.Errorf("could not parse memory duration: %s", err)
	}

	sources := framework.PluginMap[source.Source]{
		Class:   "source",
		Factory: source.NewSource,
		Require: framework.RequireName | framework.RequireKind | framework.SupportSources,
	}

	err = sources.LoadAll(config.Sources)
	if err != nil {
		return err
	}
	err = sources.ForEach(func(name string, p *framework.Plugin[source.Source]) error {
		filter, isFilter := p.Impl.(source.Filter)
		_, isRoot := p.Impl.(source.Root)
		if isFilter && !isRoot {
			if len(p.Sources) == 0 {
				return fmt.Errorf("filter %s (%s) has no dependencies", name, filter.Kind())
			}
			for _, dep := range p.Sources {
				_, err := sources.Find(dep)
				if err != nil {
					return fmt.Errorf("filter %s (%s) depends on unknown source %s", name, filter.Kind(), dep)
				}
			}
			//TODO create dependancy tree
		}
		if isRoot && !isFilter {
			if len(p.Sources) != 0 {
				return fmt.Errorf("root %s should not depend on other sources", name)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	targets := make(map[string]*Target, 0)
	for _, targetConfig := range config.Targets {
		_, ok := m.targets[targetConfig.Name]
		if ok {
			return fmt.Errorf("target %s already exists", targetConfig.Name)
		}

		job, err := NewTarget(targetConfig)
		if err != nil {
			return err
		}
		targets[targetConfig.Name] = job
	}

	//stop old sources
	if m.sources.Count() > 0 {
		m.Stop()
	}

	m.targets = targets
	m.scanFrequency = scanFrequency
	m.memoryDuration = memoryDuration
	m.sources = sources

	m.Start(ctx)
	//TODO  start old sources

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	return nil
}

func (m *Manager) Stop() error {
	return nil
}
