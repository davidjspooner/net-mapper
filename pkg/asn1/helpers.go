package asn1

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1reflect"
)

func RegisterBinaryCodecs() error {
	err1 := asn1go.Register()
	err2 := asn1reflect.Register()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}
