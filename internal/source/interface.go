package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type HostList []string

type Source interface {
	Kind() string
}
type Root interface {
	Source
	Discover(ctx context.Context) (HostList, error)
}
type Filter interface {
	Source
	Filter(ctx context.Context, input HostList) (HostList, error)
}

type FactoryFunc func(criteria framework.Config) (Source, error)

var factories map[string]FactoryFunc

func Register(name string, f FactoryFunc) {
	if factories == nil {
		factories = make(map[string]FactoryFunc)
	}
	factories[name] = f
}

func NewSource(kind string, args framework.Config) (Source, error) {
	if f, ok := factories[kind]; ok {
		return f(args)
	}
	supported := make([]string, 0, len(factories))
	for k := range factories {
		supported = append(supported, k)
	}

	return nil, fmt.Errorf("unknown source %s, should be one of %s", kind, strings.Join(supported, ","))
}
