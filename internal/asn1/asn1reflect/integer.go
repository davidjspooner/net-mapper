package asn1reflect

import (
	"reflect"
	"time"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1go"
)

var durationType = reflect.TypeOf((*time.Duration)(nil)).Elem()

type integerReflectHandler struct {
}

func (i *integerReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	if reflectedValue.Type() == durationType {
		return i.PackAsn1Duration(reflectedValue, params)
	}
	number := asn1go.Integer{}
	n := reflectedValue.Int()
	number.SetInt(n)
	return number.PackAsn1(params)
}
func (i *integerReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	if reflectedValue.Type() == durationType {
		return i.UnpackAsn1Duration(reflectedValue, envelope, bytes)
	}
	bits := reflectedValue.Type().Bits()
	number := asn1go.Integer{}
	err := number.UnpackAsn1(envelope, bytes)
	if err != nil {
		return err
	}
	n, err := number.GetInt(bits)
	if err != nil {
		return err
	}
	reflectedValue.SetInt(n)
	return nil
}
func (i *integerReflectHandler) PackAsn1Duration(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("integerReflectHandler.PackAsn1Duration")
}
func (i *integerReflectHandler) UnpackAsn1Duration(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	return asn1core.NewUnimplementedError("integerReflectHandler.UnpackAsn1Duration")
}
