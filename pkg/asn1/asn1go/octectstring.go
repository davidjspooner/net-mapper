package asn1go

import (
	"encoding/hex"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type OctetString []byte

func (v *OctetString) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{Tag: asn1binary.TagOctetString}, *v, nil
}
func (v *OctetString) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1binary.TagOctetString {
		return asn1error.NewUnexpectedError(asn1binary.TagOctetString, envelope.Tag, "unexpected tag")
	}
	*v = make([]byte, len(bytes))
	copy(*v, bytes)
	return nil
}
func (v *OctetString) String() string {
	return hex.EncodeToString(*v)
}
