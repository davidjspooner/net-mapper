package report

import (
	"context"
	"fmt"
	"strings"

	"github.com/davidjspooner/net-mapper/internal/framework"
	"github.com/davidjspooner/net-mapper/internal/source"
)

type Interface interface {
	Generate(ctx context.Context, hosts source.HostList) (string, error)
}

type FactoryFunc func(args framework.Config) (Interface, error)

var factories map[string]FactoryFunc

func Register(pubType string, f FactoryFunc) {
	if factories == nil {
		factories = make(map[string]FactoryFunc)
	}
	factories[pubType] = f
}

func NewReportGenerator(pubType string, args framework.Config) (Interface, error) {
	if f, ok := factories[pubType]; ok {
		return f(args)
	}
	supported := make([]string, 0, len(factories))
	for k := range factories {
		supported = append(supported, k)
	}
	return nil, fmt.Errorf("unknown report generator type %s, should be one of %s", pubType, strings.Join(supported, ","))
}
