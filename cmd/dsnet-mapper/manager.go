package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/davidjspooner/dsflow/pkg/job"
	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/genericutils"
	"github.com/davidjspooner/net-mapper/internal/publisher"
	"github.com/davidjspooner/net-mapper/internal/report"
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
	memoryDuration time.Duration

	lock    sync.RWMutex
	content map[string]string
	hosts   map[string]*Hosts
}

func NewManager() *Manager {
	return &Manager{
		content: make(map[string]string),
		hosts:   make(map[string]*Hosts),
	}
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

	err = sources.LoadAll("source", config.Sources)
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
		_, ok := targets[targetConfig.Name]
		if ok {
			return fmt.Errorf("target %s already exists", targetConfig.Name)
		}

		job, err := NewTarget(targetConfig)
		if err != nil {
			return err
		}
		targets[targetConfig.Name] = job
	}

	graph := job.NewNodeGraph()

	//add all the nodes and dependancies
	sources.ForEach(func(name string, p *framework.Plugin[source.Source]) error {
		graph.AddNode(p)
		return nil
	})

	errorList := job.ErrorList{}

	sources.ForEach(func(name string, p *framework.Plugin[source.Source]) error {
		if len(p.Sources) > 0 {
			for _, precursor := range p.Sources {
				err := graph.SetPrecursorIDs(p.ID(), "source."+precursor)
				if err != nil {
					errorList = append(errorList, err)
				}
			}
		}
		return nil
	})
	var publishers []job.Node

	for _, target := range targets {
		target.report.ForEach(func(name string, p *framework.Plugin[report.Interface]) error {
			graph.AddNode(p)
			for _, precursor := range target.source {
				err := graph.SetPrecursorIDs(p.ID(), "source."+precursor)
				if err != nil {
					errorList = append(errorList, err)
				}
			}
			return nil
		})
		target.publisher.ForEach(func(name string, p *framework.Plugin[publisher.Interface]) error {
			graph.AddNode(p)
			publishers = append(publishers, p)
			for _, precursor := range p.Reports {
				err := graph.SetPrecursorIDs(p.ID(), "report."+target.name+"."+precursor)
				if err != nil {
					errorList = append(errorList, err)
				}
			}
			return nil
		})
	}

	if len(errorList) > 0 {
		return errorList
	}

	plannedNodes, err := graph.PlanNodes(publishers...)
	if err != nil {
		return err
	}

	m.memoryDuration = memoryDuration

	go m.backgroundLoop(ctx, scanFrequency, plannedNodes)

	return nil
}

func (m *Manager) backgroundRun(ctx context.Context, nodes job.NodeDependancyOrdering) {
	log.Printf("Starting %d jobs\n", len(nodes))
	err := nodes.Run(ctx, 10, m.backgroundStep, log.Default())
	if err != nil {
		log.Printf("Finished jobs with error: %s\n", err)
		return
	}
	log.Printf("Finished jobs OK\n")
}
func (m *Manager) backgroundLoop(ctx context.Context, scanFrequency time.Duration, nodes job.NodeDependancyOrdering) {
	lastStarted := time.Now()
	m.backgroundRun(ctx, nodes)
	for {
		elapased := time.Since(lastStarted)
		sleepTime := scanFrequency - elapased
		if sleepTime < 0 {
			log.Printf("Scan took %s longer than %s\n", -sleepTime, scanFrequency)
			sleepTime = 1
		}
		sleepTime = genericutils.Max(sleepTime, time.Minute)

		log.Printf("Sleeping for %s\n", sleepTime)
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepTime):
			lastStarted = time.Now()
			m.backgroundRun(ctx, nodes)
		}
	}
}

func (m *Manager) updateHostsFromSource(sourceName string, hosts source.HostList) {
	m.lock.Lock()
	defer m.lock.Unlock()
	hostList, ok := m.hosts[sourceName]
	if !ok {
		hostList = &Hosts{
			source: sourceName,
		}
		m.hosts[sourceName] = hostList
	}
	hostList.Remember(hosts...)
}

func (m *Manager) getHostsFromSource(sourceName string) source.HostList {
	m.lock.RLock()
	defer m.lock.RUnlock()
	hostCache, ok := m.hosts[sourceName]
	if !ok {
		return nil
	}
	hosts := make(source.HostList, 0, len(hostCache.remembered))
	for host := range hostCache.remembered {
		hosts = append(hosts, host)
	}
	return hosts
}

func (m *Manager) backgroundStep(ctx context.Context, nodeWithPrecursors *job.NodeWithPrecursors) (err error) {
	log.Default().Printf("Starting %s\n", nodeWithPrecursors.ID())
	defer func() {
		if err != nil {
			log.Default().Printf("Finished %s with error: %s\n", nodeWithPrecursors.ID(), err)
		} else {
			log.Default().Printf("Finished %s OK\n", nodeWithPrecursors.ID())
		}
	}()
	switch node := nodeWithPrecursors.Node().(type) {
	case *framework.Plugin[source.Source]:
		root, ok := node.Impl.(source.Root)
		if ok {
			hosts, err := root.Discover(ctx)
			if err != nil {
				return err
			}
			m.updateHostsFromSource(node.ID(), hosts)
			return nil
		}
		filter, ok := node.Impl.(source.Filter)
		if ok {
			inputs := make(source.HostList, 0)
			for _, precursor := range nodeWithPrecursors.Precursors() {
				hosts := m.getHostsFromSource(precursor.ID())
				inputs = append(inputs, hosts...)
			}
			hosts, err := filter.Filter(ctx, inputs)
			if err != nil {
				return err
			}
			m.updateHostsFromSource(node.ID(), hosts)
			return nil
		}
	case *framework.Plugin[report.Interface]:
		inputs := make(source.HostList, 0)
		for _, precursor := range nodeWithPrecursors.Precursors() {
			hosts := m.getHostsFromSource(precursor.ID())
			inputs = append(inputs, hosts...)
		}
		content, err := node.Impl.Generate(ctx, inputs)
		if err != nil {
			return err
		}
		_ = content
		return nil
	case *framework.Plugin[publisher.Interface]:
		//TODO
		content := ""             //TODO
		generatedAt := time.Now() //TODO
		err := node.Impl.Publish(ctx, content, generatedAt)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown node type %T", nodeWithPrecursors.Node())
}
