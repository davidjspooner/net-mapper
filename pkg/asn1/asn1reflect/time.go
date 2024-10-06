package asn1reflect

import (
	"reflect"
	"time"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type timeReflectHandler struct {
}

func (s *timeReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {

	format := asn1binary.TagGeneralizedTime
	if params != nil && params.Tag != nil {
		format = *params.Tag
	}
	t := reflectedValue.Interface().(time.Time)
	switch format {
	case asn1binary.TagGeneralizedTime:
		return asn1binary.Envelope{Tag: asn1binary.TagGeneralizedTime}, []byte(t.Format("20060102150405Z0700")), nil
	case asn1binary.TagUTCTime:
		return asn1binary.Envelope{Tag: asn1binary.TagUTCTime}, []byte(t.Format("060102150405Z")), nil
	case asn1binary.TagDate, asn1binary.TagTime:
		return asn1binary.Envelope{}, nil, asn1error.NewUnimplementedError("structReflectHandler.PackAsn1Time")
	default:
		return asn1binary.Envelope{}, nil, asn1error.NewUnimplementedError("unsupported time format %s", format).TODO()
	}
}
func (s *timeReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	switch envelope.Tag {
	case asn1binary.TagGeneralizedTime:
		t, err := time.Parse("20060102150405Z0700", string(bytes))
		if err != nil {
			return err
		}
		reflectedValue.Set(reflect.ValueOf(t))
		return nil
	case asn1binary.TagUTCTime:
		t, err := time.Parse("060102150405Z", string(bytes))
		if err != nil {
			return err
		}
		reflectedValue.Set(reflect.ValueOf(t))
		return nil
	case asn1binary.TagDate, asn1binary.TagTime:
		return asn1error.NewUnimplementedError("structReflectHandler.PackAsn1Time")
	default:
		return asn1error.NewUnimplementedError("unsupported time format %s", envelope.Tag).TODO()
	}
}
