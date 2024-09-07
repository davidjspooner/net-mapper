package source

import (
	"context"
	"fmt"
	"net"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Static struct {
	hosts HostList
	cidr  []net.IPNet
	size  int
}

var _ Root = (*Static)(nil)

func init() {
	Register("static", newStaticRoot)
}

func newStaticRoot(args framework.Config) (Source, error) {
	static := &Static{}

	err := framework.CheckKeys(args, "cidr", "hosts")
	if err != nil {
		return nil, err
	}

	cidranys, err := framework.GetArg(args, "cidr", []interface{}{})
	if err != nil {
		return nil, err
	}
	for _, cidrany := range cidranys {
		cidrString, ok := cidrany.(string)
		if !ok {
			return nil, fmt.Errorf("invalid cidr %v", cidrany)
		}
		_, cdir, err := net.ParseCIDR(cidrString)
		if err != nil {
			return nil, fmt.Errorf("invalid cdir %s: %s", cidrString, err)
		}

		ones, _ := cdir.Mask.Size() //TODO adapt this for IPV6
		if ones < 24 {
			return nil, fmt.Errorf("cdir %s is larger than /24", cidrString)
		}

		static.cidr = append(static.cidr, *cdir)
	}

	hosts, _ := framework.GetArg(args, "hosts", []any{})
	for _, hostany := range hosts {
		host, ok := hostany.(string)
		if !ok {
			return nil, fmt.Errorf("invalid host %v", hostany)
		}
		static.hosts = append(static.hosts, host)
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

func (static *Static) Discover(ctx context.Context) (HostList, error) {
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

func (static *Static) Kind() string {
	return "static"
}
