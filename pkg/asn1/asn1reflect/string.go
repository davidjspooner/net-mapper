package asn1reflect

import (
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

type stringReflectHandler struct {
}

func (s *stringReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	str := asn1go.String{
		Envelope: asn1binary.Envelope{Tag: asn1binary.TagUTF8String},
		Elem:     reflectedValue.String(),
	}
	env, bytes, err := str.PackAsn1(params)
	if err != nil {
		return env, nil, err
	}
	return env, bytes, nil
}
func (s *stringReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	str := &asn1go.String{
		Envelope: envelope,
	}
	err := str.UnpackAsn1(envelope, bytes)
	if err != nil {
		return err
	}
	reflectedValue.SetString(str.Elem)
	return nil
}
