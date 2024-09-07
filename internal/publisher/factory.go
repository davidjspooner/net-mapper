package publisher

import (
	"context"
	"fmt"
	"strings"
	"time"
	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Interface interface {
	Publish(ctx context.Context, report string, generated time.Time) error
}

type FactoryFunc func(args framework.Config) (Interface, error)

var factories map[string]FactoryFunc

func Register(pubType string, f FactoryFunc) {
	if factories == nil {
		factories = make(map[string]FactoryFunc)
	}
	factories[pubType] = f
}

func NewPublisher(pubType string, args framework.Config) (Interface, error) {
	if f, ok := factories[pubType]; ok {
		return f(args)
	}
	supported := make([]string, 0, len(factories))
	for k := range factories {
		supported = append(supported, k)
	}
	return nil, fmt.Errorf("unknown publisher type %s, should be one of %s", pubType, strings.Join(supported, ","))
}
