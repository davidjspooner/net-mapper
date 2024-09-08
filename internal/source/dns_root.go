package source

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type dnsZoneRoot struct {
	server     []string
	zone       []string
	regex      []regexp.Regexp
	types      []string
	tsigName   string
	tsigSecret string
	tsigAlgo   string
}

var _ Root = (*dnsZoneRoot)(nil)

func init() {
	Register("dns", newDnsZoneRoot)
}

func newDnsZoneRoot(args framework.Config) (Source, error) {
	dzs := &dnsZoneRoot{}

	err := framework.CheckFields(args, "server", "zone", "regex", "types", "tsig_name", "tsig_secret", "tsig_algorithm", "interval")
	if err != nil {
		return nil, err
	}
	dzs.server, err = framework.ConsumeArg[[]string](args, "server")
	if err != nil {
		return nil, err
	}

	dzs.zone, err = framework.ConsumeArg[[]string](args, "zone")
	if err != nil {
		return nil, err
	}

	regex, err := framework.ConsumeArg[[]string](args, "regex")
	if err != nil {
		return nil, err
	}
	for _, r := range regex {
		re, err := regexp.Compile(r)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %s: %s", r, err)
		}
		dzs.regex = append(dzs.regex, *re)
	}

	dzs.types, err = framework.ConsumeOptionalArg(args, "types", []string{"A", "AAAA", "CNAME"})
	if err != nil {
		return nil, err
	}
	for _, t := range dzs.types {
		if !slices.Contains([]string{"A", "AAAA", "CNAME"}, t) {
			return nil, fmt.Errorf("invalid type %s ( need to be one of A,AAAA,CNAME)", t)
		}
	}

	dzs.tsigName, err = framework.ConsumeOptionalArg(args, "tsig_name", "")
	if err != nil {
		return nil, err
	}
	dzs.tsigSecret, err = framework.ConsumeOptionalArg(args, "tsig_secret", "")
	if err != nil {
		return nil, err
	}
	dzs.tsigAlgo, err = framework.ConsumeOptionalArg(args, "tsig_algorithm", "")
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

func (d *dnsZoneRoot) Discover(ctx context.Context) (HostList, error) {
	return nil, fmt.Errorf("dns zone scanner not implemented")
}

func (d *dnsZoneRoot) Kind() string {
	return "dns"
}
