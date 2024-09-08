package source

import (
	"context"
	"fmt"
	"net"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type staticRoot struct {
	hosts HostList
	cidr  []net.IPNet
	size  int
}

var _ Root = (*staticRoot)(nil)

func init() {
	Register("static", newStaticRoot)
}

func newStaticRoot(args framework.Config) (Source, error) {
	static := &staticRoot{}

	err := framework.CheckFields(args, "cidr", "hosts")
	if err != nil {
		return nil, err
	}

	cidranys, err := framework.ConsumeOptionalArg(args, "cidr", []string{})
	if err != nil {
		return nil, err
	}
	for _, cidrString := range cidranys {
		_, cdir, err := net.ParseCIDR(cidrString)
		if err != nil {
			return nil, fmt.Errorf("invalid cdir %q: %s", cidrString, err)
		}

		ones, _ := cdir.Mask.Size() //TODO adapt this for IPV6
		if ones < 24 {
			return nil, fmt.Errorf("cdir %q is larger than /24", cidrString)
		}

		static.cidr = append(static.cidr, *cdir)
	}

	static.hosts, err = framework.ConsumeOptionalArg(args, "hosts", []string{})
	if err != nil {
		return nil, err
	}

	if len(static.cidr) == 0 && len(static.hosts) == 0 {
		return nil, fmt.Errorf("no hosts or cidr specified")
	}

	return static, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (static *staticRoot) Discover(ctx context.Context) (HostList, error) {
	h := make(HostList, static.size)
	for _, cdir := range static.cidr {
		for ip := cdir.IP.Mask(cdir.Mask); cdir.Contains(ip); incIP(ip) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			h = append(h, ip.String())
		}
	}
	h = append(h, static.hosts...)

	static.size = len(h)
	return h, nil
}

func (static *staticRoot) Kind() string {
	return "static"
}
