package asn1go

import (
	"encoding/hex"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

//--------------------------------------------------------------------------------------------

type Integer []byte

func (v *Integer) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{Tag: asn1core.TagInteger}, *v, nil
}
func (v *Integer) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagInteger {
		return asn1core.NewUnexpectedError(asn1core.TagInteger, envelope.Tag, "unexpected tag")
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
	if (*v)[0] == 0xFF { //supeflous leading negative sign
		return L*8 - 1
	}
	if (*v)[0] == 0x00 { //supeflous leading zero
		return L*8 - 1
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
		return 0, asn1core.NewErrorf("too many bits for int64")
	}
	if bits%8 != 0 {
		return 0, asn1core.NewErrorf("bits must be a multiple of 8")
	}
	if len(*v) == 0 {
		return 0, nil
	}
	n := int64(0)
	negate := (*v)[0]&0x80 != 0

	bitsRead := 0
	for i := len(*v) - 1; i >= 0; i-- {
		if bitsRead >= bits && (negate && (*v)[i] != 0xFF || !negate && (*v)[i] != 0) {
			return 0, asn1core.NewErrorf("integer is too large for %d bits", bits)
		}
		n = n<<8 + int64((*v)[i])
		bitsRead += 8
	}
	if negate {
		n = -n
	}
	return n, nil
}
func (v *Integer) SetInt(value int64) {
	negate := false
	if value < 0 {
		negate = true
		value = -value
	}
	buffer := make([]byte, 0, 9)
	if value == 0 {
		*v = append(buffer, 0)
		return
	}
	for value > 0 {
		buffer = append(buffer, byte(value&0xFF))
		value >>= 8
	}
	vLen := len(buffer)
	lastByte := (buffer)[vLen-1]
	if !negate && lastByte&0x80 != 0 {
		buffer = append(buffer, 0)
	}
	*v = make([]byte, len(buffer))
	//reverse the bytes
	for i := 0; i < len(buffer); i++ {
		(*v)[i] = buffer[len(buffer)-1-i]
	}
}
