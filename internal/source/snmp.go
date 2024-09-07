package source

import (
	"context"
	"fmt"
	"slices"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type Snmp struct {
	community string
	version   string
	oid       string
}

var _ Filter = (*Snmp)(nil)

func ValidateOID(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("empty string")
	}
	if len(s) > 128 {
		return fmt.Errorf("string too long")
	}
	for i, c := range s {
		if i == 0 && !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return fmt.Errorf("first character must be a letter")
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.') {
			return fmt.Errorf("invalid character %c", c)
		}
	}
	return nil
}

func NewSnmp(args framework.Config) (Source, error) {
	s := &Snmp{}

	err := framework.CheckKeys(args, "community", "version", "oid")
	if err != nil {
		return nil, err
	}

	s.community, err = framework.GetArg(args, "community", "")
	if err != nil {
		return nil, err
	}
	if s.community == "" {
		return nil, fmt.Errorf("community is empty")
	}
	s.version, err = framework.GetArg(args, "version", "v2c")
	if err != nil {
		return nil, err
	}
	if !slices.Contains([]string{"v1", "v2c", "v3"}, s.version) {
		return nil, fmt.Errorf("invalid version %s ( need to be one of v1,v2c,v3)", s.version)
	}
	s.oid, err = framework.GetArg(args, "oid", "")
	if err != nil {
		return nil, err
	}
	err = ValidateOID(s.oid)
	if err != nil {
		return nil, fmt.Errorf("oid %s is invalid: %s", s.oid, err)
	}

	return s, nil
}

func (h *Snmp) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("snmp condition not implemented")
}

func (h *Snmp) Kind() string {
	return "snmp"
}
