package asn1go

import (
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type String struct {
	asn1binary.Envelope
	Elem string
}

func (v *String) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	e := v.Envelope
	if params != nil {
		if params.Tag != nil {
			e.Tag = *params.Tag
		}
		if params.Class != nil {
			e.Class = *params.Class
		}
	}
	switch e.Tag {
	case asn1core.TagUTF8String:
		return e, []byte(v.Elem), nil
	case asn1core.TagBMPString:
		b := make([]byte, 0, len(v.Elem)*2)
		for _, r := range v.Elem {
			b = append(b, byte(r>>8), byte(r))
		}
		return e, b, nil
	case asn1core.TagPrintableString:
		b := []byte(v.Elem)
		err := asn1binary.PrintableStringValidator.ValidateBytes(b)
		if err != nil {
			return asn1binary.Envelope{}, nil, err
		}
		return e, b, nil
	case asn1core.TagIA5String:
		b := []byte(v.Elem)
		err := asn1binary.IA5StringValidator.ValidateBytes(b)
		if err != nil {
			return asn1binary.Envelope{}, nil, err
		}
		return e, b, nil
	}
	return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("cannot pack string as tag %s", e.Tag)
}
func (v *String) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	switch envelope.Tag {
	case asn1core.TagUTF8String:
		v.Elem = string(bytes)
	case asn1core.TagPrintableString:
		err := asn1binary.PrintableStringValidator.ValidateBytes(bytes)
		if err != nil {
			return err
		}
		v.Elem = string(bytes)
	case asn1core.TagOctetString:
		v.Elem = string(bytes)
	case asn1core.TagIA5String:
		err := asn1binary.IA5StringValidator.ValidateBytes(bytes)
		if err != nil {
			return err
		}
		v.Elem = string(bytes)
	case asn1core.TagBMPString:
		if len(bytes)%2 != 0 {
			return asn1core.NewErrorf("BMPString length is not even")
		}
		s := make([]rune, len(bytes)/2)
		for i := 0; i < len(bytes)/2; i++ {
			s[i] = rune(uint16(bytes[i*2])<<8 + uint16(bytes[i*2+1]))
		}
		v.Elem = string(s)
	default:
		return asn1core.NewUnimplementedError("cannot unpack string from tag %s", envelope.Tag)
	}
	return nil
}
func (v *String) String() string {
	return v.Elem
}
