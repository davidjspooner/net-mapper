package asn1go

import (
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
)

func packerForGoType(i any) (asn1binary.Packer, error) {
	packer, ok := i.(asn1binary.Packer)
	if ok {
		return packer, nil
	}
	return nil, fmt.Errorf("type %T does not implement asn1binary.Packer", i)
}

func unpackerForGoType(i any) (asn1binary.Unpacker, error) {
	unpacker, ok := i.(asn1binary.Unpacker)
	if ok {
		return unpacker, nil
	}
	return nil, fmt.Errorf("type %T does not implement asn1binary.Unpacker", i)
}

func Register() error {
	return asn1binary.RegisterProviderFuncs(0, "go", packerForGoType, unpackerForGoType)
}
