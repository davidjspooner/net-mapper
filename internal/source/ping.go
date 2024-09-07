package source

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Ping struct {
}

var _ Filter = (*Ping)(nil)

func init() {
	Register("ping", newPingFilter)
}

func newPingFilter(args framework.Config) (Source, error) {
	h := &Ping{}

	err := framework.CheckKeys(args)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Ping) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("Ping condition not implemented")
}

func (h *Ping) Kind() string {
	return "ping"
}
