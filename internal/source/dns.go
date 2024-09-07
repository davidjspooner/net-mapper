package source

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type DnsZone struct {
	server     []string
	zone       []string
	regex      []regexp.Regexp
	types      []string
	tsigName   string
	tsigSecret string
	tsigAlgo   string
}

var _ Root = (*DnsZone)(nil)

func init() {
	Register("dns", newDnsZoneRoot)
}

func newDnsZoneRoot(args framework.Config) (Source, error) {
	dzs := &DnsZone{}

	err := framework.CheckKeys(args, "server", "zone", "regex", "types", "tsig_name", "tsig_secret", "tsig_algorithm", "interval")
	if err != nil {
		return nil, err
	}
	dzs.server, err = framework.GetArg(args, "server", []string{})
	if err != nil {
		return nil, err
	}
	if len(dzs.server) == 0 {
		return nil, fmt.Errorf("no servers specified")
	}

	dzs.zone, err = framework.GetArg(args, "zone", []string{})
	if err != nil {
		return nil, err
	}
	if len(dzs.zone) == 0 {
		return nil, fmt.Errorf("no zones specified")
	}

	regex, err := framework.GetArg(args, "regex", []string{})
	if err != nil {
		return nil, err
	}
	if len(regex) == 0 {
		return nil, fmt.Errorf("no regex specified")
	}
	for _, r := range regex {
		re, err := regexp.Compile(r)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %s: %s", r, err)
		}
		dzs.regex = append(dzs.regex, *re)
	}

	dzs.types, err = framework.GetArg(args, "types", []string{"A", "AAAA", "CNAME"})
	if err != nil {
		return nil, err
	}
	for _, t := range dzs.types {
		if !slices.Contains([]string{"A", "AAAA", "CNAME"}, t) {
			return nil, fmt.Errorf("invalid type %s ( need to be one of A,AAAA,CNAME)", t)
		}
	}

	dzs.tsigName, err = framework.GetArg(args, "tsig_name", "")
	if err != nil {
		return nil, err
	}
	dzs.tsigSecret, err = framework.GetArg(args, "tsig_secret", "")
	if err != nil {
		return nil, err
	}
	dzs.tsigAlgo, err = framework.GetArg(args, "tsig_algorithm", "")
	if err != nil {
		return nil, err
	}

	if dzs.tsigSecret != "" || dzs.tsigName != "" || dzs.tsigAlgo != "" {
		if dzs.tsigName == "" {
			return nil, fmt.Errorf("tsigName is empty")
		}
		if dzs.tsigSecret == "" {
			return nil, fmt.Errorf("tsigSecret is empty")
		}
		if dzs.tsigAlgo == "" {
			return nil, fmt.Errorf("tsigAlgo is empty")
		}
	}

	return dzs, nil
}

func (d *DnsZone) Discover(ctx context.Context) (HostList, error) {
	return nil, fmt.Errorf("dns zone scanner not implemented")
}

func (d *DnsZone) Kind() string {
	return "dns"
}
