package source

import (
	"context"
	"fmt"
	"slices"

	"github.com/davidjspooner/net-mapper/internal/framework"
)

type snmpFilter struct {
	community string
	version   string
	oid       []string
}

var _ Filter = (*snmpFilter)(nil)

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

func init() {
	Register("snmp", newSnmpFilter)
}

func newSnmpFilter(args framework.Config) (Source, error) {
	s := &snmpFilter{}

	err := framework.CheckFields(args, "community", "version", "oid")
	if err != nil {
		return nil, err
	}

	s.community, err = framework.ConsumeOptionalArg(args, "community", "")
	if err != nil {
		return nil, err
	}
	if s.community == "" {
		return nil, fmt.Errorf("community is empty")
	}
	s.version, err = framework.ConsumeOptionalArg(args, "version", "v2c")
	if err != nil {
		return nil, err
	}
	if !slices.Contains([]string{"v1", "v2c", "v3"}, s.version) {
		return nil, fmt.Errorf("invalid version %s ( need to be one of v1,v2c,v3)", s.version)
	}
	s.oid, err = framework.ConsumeOptionalArg(args, "oid", []string{})
	if err != nil {
		return nil, err
	}
	for _, o := range s.oid {
		err = ValidateOID(o)
		if err != nil {
			return nil, fmt.Errorf("oid %q is invalid: %s", o, err)
		}
	}

	return s, nil
}

func (h *snmpFilter) Filter(ctx context.Context, input HostList) (HostList, error) {
	return nil, fmt.Errorf("snmp condition not implemented")
}

func (h *snmpFilter) Kind() string {
	return "snmp"
}
