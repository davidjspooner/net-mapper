package asn1go

import (
	"bytes"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type Sequence[T any] struct {
	asn1binary.Envelope
	Elem []T
}

func (v *Sequence[T]) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	v.Envelope.Tag = asn1core.TagSequence
	err := v.Envelope.UpdatePer(params)
	if err != nil {
		return asn1binary.Envelope{}, nil, err
	}

	b := bytes.Buffer{}
	var tmp asn1binary.Value
	for n, elem := range v.Elem {
		packer, err := asn1binary.GetPackerFor(elem)
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1core.NewErrorf("packing sequence element #%d", n).WithCause(err)
		}
		tmp.Envelope, tmp.Bytes, err = packer.PackAsn1(params)
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1core.NewErrorf("packing sequence element #%d", n).WithCause(err)
		}
		chunk, err := tmp.Marshal()
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1core.NewErrorf("marshalling sequence element #%d", n).WithCause(err)
		}
		b.Write(chunk)
	}
	return v.Envelope, b.Bytes(), nil
}

func (v *Sequence[T]) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	v.Envelope = envelope
	n := 0
	for len(bytes) > 0 {
		var tmp asn1binary.Value
		tail, err := tmp.Unmarshal(bytes)
		if err != nil {
			return asn1core.NewErrorf("unmarshalling sequence element #%d", n).WithCause(err)
		}
		var elem T
		unpacker, err := asn1binary.GetUnpackerFor(elem)
		if err != nil {
			return asn1core.NewErrorf("unpacking sequence element #%d", n).WithCause(err)
		}
		err = unpacker.UnpackAsn1(tmp.Envelope, tmp.Bytes)
		if err != nil {
			return asn1core.NewErrorf("unpacking sequence element #%d", n).WithCause(err)
		}
		v.Elem = append(v.Elem, elem)
		bytes = tail
		n++
	}

	return nil
}
