package framework

import (
	"fmt"
)

type Plugin[T any] struct {
	Name    string
	Kind    string
	Sources []string
	Reports []string
	Impl    T
}

type Required int

const (
	RequireName Required = 1 << iota
	SupportName
	RequireKind
	SupportKind
	SupportSources
	RequireReports
)

type PluginMap[T any] struct {
	plugins map[string]*Plugin[T]
	Class   string
	Factory func(string, Config) (T, error)
	Require Required
}

func (pm *PluginMap[T]) Load(spec Config) error {
	if pm.plugins == nil {
		pm.plugins = make(map[string]*Plugin[T], 1)
	}

	var err error
	var name, kind string

	if pm.Require&RequireName != 0 {
		name, err = ConsumeArg[string](spec, "name")
		if err != nil {
			return err
		}
	} else if pm.Require&SupportName != 0 {
		name, err = ConsumeOptionalArg[string](spec, "name", "")
		if err != nil {
			return err
		}
		if name == "" {
			name = fmt.Sprintf("#%d", len(pm.plugins)+1)
		}
	} else {
		name = fmt.Sprintf("#%d", len(pm.plugins)+1)
	}
	if _, ok := pm.plugins[name]; ok {
		return fmt.Errorf("%s %s already registered", pm.Class, name)
	}
	if pm.Require&RequireKind != 0 {
		kind, err = ConsumeArg[string](spec, "kind")
		if err != nil {
			return err
		}
	} else if pm.Require&SupportKind != 0 {
		kind, err = ConsumeOptionalArg[string](spec, "kind", "default")
		if err != nil {
			return err
		}
	}

	var sources, reports []string

	if pm.Require&SupportSources != 0 {
		sources, err = ConsumeOptionalArg(spec, "source", []string{})
		if err != nil {
			return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
		}
	}

	if pm.Require&RequireReports != 0 {
		reports, err = ConsumeOptionalArg(spec, "report", []string{})
		if err != nil {
			return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
		}
	}

	impl, err := pm.Factory(kind, spec)
	if err != nil {
		return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
	}

	pm.plugins[name] = &Plugin[T]{
		Name:    name,
		Kind:    kind,
		Impl:    impl,
		Sources: sources,
		Reports: reports,
	}

	return nil
}

func (pm *PluginMap[T]) LoadAll(specs []Config) error {
	for _, spec := range specs {
		if err := pm.Load(spec); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PluginMap[T]) Find(name string) (*Plugin[T], error) {
	if pm.plugins == nil {
		return nil, fmt.Errorf("no %s registered", pm.Class)
	}
	if p, ok := pm.plugins[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("no %s named %s", pm.Class, name)
}

func (pm *PluginMap[T]) ForEach(fn func(string, *Plugin[T]) error) error {
	for name, p := range pm.plugins {
		if err := fn(name, p); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PluginMap[T]) Names() []string {
	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

func (pm *PluginMap[T]) Count() int {
	return len(pm.plugins)
}
