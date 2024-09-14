package asn1go

import (
	"encoding/hex"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type OctetString []byte

func (v *OctetString) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{Tag: asn1core.TagOctetString}, *v, nil
}
func (v *OctetString) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagOctetString {
		return asn1core.NewUnexpectedError(asn1core.TagOctetString, envelope.Tag, "unexpected tag")
	}
	*v = make([]byte, len(bytes))
	copy(*v, bytes)
	return nil
}
func (v *OctetString) String() string {
	return hex.EncodeToString(*v)
}
