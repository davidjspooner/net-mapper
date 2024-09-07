package framework

import (
	"fmt"
)

type Plugin[T any] struct {
	Name    string
	Kind    string
	Sources []string
	Report  string
	Impl    T
}

type PluginMap[T any] struct {
	plugins map[string]*Plugin[T]
	Class   string
	Factory func(string, Config) (T, error)
}

func (pm *PluginMap[T]) Load(spec Config) error {
	if pm.plugins == nil {
		pm.plugins = make(map[string]*Plugin[T], 1)
	}

	name, err := GetArg(spec, "name", "")
	if err != nil {
		return err
	}
	kind, err := GetArg(spec, "kind", "")
	if err != nil {
		return err
	}

	if _, ok := pm.plugins[name]; ok {
		return fmt.Errorf("%s %s already registered", pm.Class, name)
	}

	impl, err := pm.Factory(kind, spec)
	if err != nil {
		return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
	}

	sources, err := GetArg(spec, "sources", []string{})
	if err != nil {
		return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
	}
	report, err := GetArg(spec, "report", "")
	if err != nil {
		return fmt.Errorf("failed to create %s %s : %s", pm.Class, name, err)
	}

	delete(spec, "name")
	delete(spec, "kind")
	delete(spec, "sources")
	delete(spec, "report")

	pm.plugins[name] = &Plugin[T]{
		Name:    name,
		Kind:    kind,
		Impl:    impl,
		Sources: sources,
		Report:  report,
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
