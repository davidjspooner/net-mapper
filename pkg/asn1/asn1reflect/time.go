package asn1reflect

import (
	"reflect"
	"time"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

type timeReflectHandler struct {
}

func (s *timeReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {

	format := asn1core.TagGeneralizedTime
	if params != nil && params.Tag != nil {
		format = *params.Tag
	}
	t := reflectedValue.Interface().(time.Time)
	switch format {
	case asn1core.TagGeneralizedTime:
		return asn1binary.Envelope{Tag: asn1core.TagGeneralizedTime}, []byte(t.Format("20060102150405Z0700")), nil
	case asn1core.TagUTCTime:
		return asn1binary.Envelope{Tag: asn1core.TagUTCTime}, []byte(t.Format("060102150405Z")), nil
	case asn1core.TagDate, asn1core.TagTime:
		return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("structReflectHandler.PackAsn1Time")
	default:
		return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("unsupported time format %s", format).MaybeLater()
	}
}
func (s *timeReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	switch envelope.Tag {
	case asn1core.TagGeneralizedTime:
		t, err := time.Parse("20060102150405Z0700", string(bytes))
		if err != nil {
			return err
		}
		reflectedValue.Set(reflect.ValueOf(t))
		return nil
	case asn1core.TagUTCTime:
		t, err := time.Parse("060102150405Z", string(bytes))
		if err != nil {
			return err
		}
		reflectedValue.Set(reflect.ValueOf(t))
		return nil
	case asn1core.TagDate, asn1core.TagTime:
		return asn1core.NewUnimplementedError("structReflectHandler.PackAsn1Time")
	default:
		return asn1core.NewUnimplementedError("unsupported time format %s", envelope.Tag).MaybeLater()
	}
}
