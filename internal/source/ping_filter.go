package source

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type pingFilter struct {
}

var _ Filter = (*pingFilter)(nil)

func init() {
	Register("ping", newPingFilter)
}

func newPingFilter(args framework.Config) (Source, error) {
	h := &pingFilter{}

	err := framework.CheckFields(args)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *pingFilter) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("ping condition not implemented")
}

func (h *pingFilter) Kind() string {
	return "ping"
}
