package source

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Upnp struct {
}

var _ Root = (*Upnp)(nil)

func init() {
	Register("upnp", newUpnpRoot)
}

func newUpnpRoot(args framework.Config) (Source, error) {
	us := &Upnp{}

	return us, nil
}

func (us *Upnp) Discover(ctx context.Context) (HostList, error) {
	return nil, fmt.Errorf("upnp scanner not implemented")
}

func (us *Upnp) Kind() string {
	return "upnp"
}
