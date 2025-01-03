package asn1go

import (
	"encoding/hex"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

//--------------------------------------------------------------------------------------------

type Integer []byte

func (v *Integer) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{Tag: asn1binary.TagInteger}, *v, nil
}
func (v *Integer) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Class == asn1binary.ClassUniversal && envelope.Tag != asn1binary.TagInteger {
		return asn1error.NewUnexpectedError(asn1binary.TagInteger, envelope.Tag, "unexpected tag")
	}
	*v = make([]byte, len(bytes))
	copy(*v, bytes)
	return nil
}
func (v *Integer) BitSize() int {
	L := len(*v)
	if L == 0 {
		return 8 //effectivly an empty byte
	}
	switch (*v)[0] {
	case 0x00: //supeflous leading zero
		L--
	case 0xFF: //supeflous leading negative sign
		L--
	}
	return L * 8
}

func (v *Integer) String() string {
	n, err := v.GetInt(64)
	if err != nil {
		return "0x" + hex.EncodeToString(*v)
	} else {
		return strconv.FormatInt(n, 10)
	}
}
func (v *Integer) GetInt(bits int) (int64, error) {
	if bits > 64 {
		return 0, asn1error.NewErrorf("too many bits for int64")
	}
	if bits%8 != 0 {
		return 0, asn1error.NewErrorf("bits must be a multiple of 8")
	}
	if len(*v) == 0 {
		return 0, nil
	}
	n := int64(0)
	negate := (*v)[0]&0x80 != 0

	bitsRead := 0
	//for i := len(*v) - 1; i >= 0; i-- {
	for i := 0; i < len(*v); i++ {
		if bitsRead >= bits && ((negate && (*v)[i] != 0xFF) || (!negate && (*v)[i] != 0)) {
			return 0, asn1error.NewErrorf("integer is too large for %d bits", bits)
		}
		n = n<<8 + int64((*v)[i])
		bitsRead += 8
	}
	if negate {
		n <<= (64 - bitsRead)
		n >>= (64 - bitsRead)
	}
	return n, nil
}

func (v *Integer) SetInt(value int64) {
	*v = make([]byte, 8)
	negate := false
	if value < 0 {
		negate = true
	}
	for i := 7; i >= 0; i-- {
		(*v)[i] = byte(value & 0xFF)
		value >>= 8
	}

	trim := byte(0x00)
	if negate {
		trim = 0xFF
	}
	for len(*v) > 1 && (*v)[0] == trim && ((*v)[1]&0x80) == trim&0x80 {
		*v = (*v)[1:]
	}
}
