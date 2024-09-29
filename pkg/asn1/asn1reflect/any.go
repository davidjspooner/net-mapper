package asn1reflect

import (
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

type anyReflectHandler struct {
}

func (sfh *anyReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	if reflectedValue.IsNil() {
		return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("packing a NULL value for %s", reflectedValue.Type().String()).TODO()
	}
	realValue := reflectedValue.Elem()
	packer, err := getPackerForReflectedValue(realValue)
	if err != nil {
		return asn1binary.Envelope{}, nil, err
	}
	env, bytes, err := packer.PackAsn1(params)
	if err != nil {
		return asn1binary.Envelope{}, nil, err
	}
	return env, bytes, nil
}
func (sfh *anyReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	if reflectedValue.IsNil() {
		if reflectedValue.NumMethod() == 0 {
			x := &asn1go.Any{}
			err := x.UnpackAsn1(envelope, bytes)
			if err != nil {
				return err
			}
			reflectedValue.Set(reflect.ValueOf(x))
		}
		return asn1core.NewErrorf("unpacking a NULL value for %s", reflectedValue.Type().String())
	}
	realValue := reflectedValue.Elem()
	unpacker, err := getUnpackerForReflectedValue(realValue)
	if err != nil {
		return err
	}
	return unpacker.UnpackAsn1(envelope, bytes)
}
