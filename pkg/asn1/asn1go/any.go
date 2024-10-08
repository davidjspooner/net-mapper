package asn1go

import (
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type Any struct {
	asn1binary.Envelope
	Elem any
}

func (v *Any) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	e := v.Envelope
	if params != nil {
		if params.Class != nil {
			e.Class = *params.Class
		}
		if params.Tag != nil {
			e.Tag = *params.Tag
		}
	}
	if v.Elem == nil {
		return e, nil, nil
	}
	packer, err := asn1binary.GetPackerFor(v.Elem)
	if err != nil {
		return e, nil, err
	}
	_, bytes, err := packer.PackAsn1(params)
	if err != nil {
		return e, nil, err
	}
	return e, bytes, nil
}
func (v *Any) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	v.Envelope = envelope
	if len(bytes) == 0 {
		v.Elem = nil
		return nil
	}

	switch envelope.Tag {
	case asn1binary.TagBoolean:
		v.Elem = new(bool)
	case asn1binary.TagOctetString:
		v.Elem = new(OctetString)
	case asn1binary.TagIA5String, asn1binary.TagPrintableString, asn1binary.TagUTF8String:
		v.Elem = new(string)
	case asn1binary.TagNull:
		v.Elem = new(Null)
	case asn1binary.TagSequence:
		v.Elem = new(Sequence[Any])
	default:
		return asn1error.NewUnimplementedError("Any.UnpackAsn1 tag %s", envelope.Tag)
	}
	unpacker, err := asn1binary.GetUnpackerFor(v.Elem)
	if err != nil {
		return err
	}
	err = unpacker.UnpackAsn1(envelope, bytes)
	if err != nil {
		return err
	}
	return nil
}

func (v *Any) String() string {
	return fmt.Sprint(v.Elem)
}
