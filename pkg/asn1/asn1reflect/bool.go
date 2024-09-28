package asn1reflect

import (
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

type booleanReflectHandler struct {
}

func (b *booleanReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	bValue := reflectedValue.Bool()
	if bValue {
		return asn1binary.Envelope{Tag: asn1core.TagBoolean}, []byte{1}, nil
	} else {
		return asn1binary.Envelope{Tag: asn1core.TagBoolean}, []byte{0}, nil
	}
}
func (b *booleanReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	if len(bytes) != 1 {
		return asn1core.NewUnexpectedError(1, len(bytes), "boolean value").WithUnits("bytes")
	}
	if envelope.Tag != asn1core.TagBoolean {
		return asn1core.NewUnexpectedError(asn1core.TagBoolean, envelope.Tag, "unexpected tag")
	}
	reflectedValue.SetBool(bytes[0] != 0)
	return nil
}
