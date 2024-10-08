package asn1binary

import (
	"io"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type Value struct {
	Envelope
	Bytes []byte
}

func (v *Value) Marshal() ([]byte, error) {

	encodedClassAndBytes := (int(v.Class) << 6) | (int(v.Tag) & 0x3F)

	length := len(v.Bytes)
	if length < 128 {
		b := make([]byte, 2+length)
		b[0] = byte(encodedClassAndBytes)
		b[1] = byte(length)
		copy(b[2:], v.Bytes)
		return b, nil
	}
	var encodedLength [6]byte
	byteCount := length >> 8
	if length&0xFF != 0 {
		byteCount++
	}
	encodedLength[0] = byte(0x80 | byteCount)
	for i := 0; i < byteCount; i++ {
		encodedLength[byteCount-i] = byte(length >> (i * 8)) //this may need to be reversed - untested
	}
	b := make([]byte, 2+byteCount+length)
	b[0] = byte(encodedClassAndBytes)
	copy(b[1:], encodedLength[:byteCount+1])
	copy(b[2+byteCount:], v.Bytes)
	return b, nil
}

func (v *Value) Unmarshal(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, asn1error.NewUnexpectedError[int](2, len(data), "envelope truncated").WithUnits("byte(s)")
	}
	encodedClassAndBytes := data[0]
	v.Class = Class(encodedClassAndBytes >> 6)
	v.Tag = Tag(encodedClassAndBytes & 0x3F)
	length := int(data[1])
	if length < 128 {
		if len(data) < 2+length {
			return nil, asn1error.NewUnexpectedError[int](2+length, len(data), "frame truncated").WithUnits("byte(s)")
		}
		v.Bytes = data[2 : 2+length]
		return data[2+length:], nil
	}
	byteCount := length & 0x7F
	if len(data) < 2+byteCount {
		return nil, asn1error.NewUnexpectedError[int](2+byteCount, len(data), "envelope(long) truncated").WithUnits("byte(s)")
	}
	length = 0
	for i := 0; i < byteCount; i++ {
		length |= int(data[1+byteCount-i]) << (i * 8)
	}
	if len(data) < 2+byteCount+length {
		return nil, asn1error.NewUnexpectedError[int](2+length, len(data), "frame(long) truncated").WithUnits("byte(s)")
	}
	v.Bytes = data[2+byteCount : 2+byteCount+length]
	return data[2+byteCount+length:], nil
}

func (v *Value) ReadFrom(r io.Reader) (totalRead int64, err error) {

	var envelope [2]byte

	var chunkRead int
	chunkRead, err = r.Read(envelope[:])
	totalRead = int64(chunkRead)
	if err != nil {
		return totalRead, err
	}
	if chunkRead != 2 {
		return totalRead, asn1error.NewUnexpectedError[int](2, chunkRead, "envelope truncated").WithUnits("byte(s)")
	}
	v.Class = Class(envelope[0] >> 6)
	v.Tag = Tag(envelope[0] & 0x3F)
	if envelope[1] < 128 {
		v.Bytes = make([]byte, envelope[1])
		chunkRead, err = r.Read(v.Bytes)
		totalRead += int64(chunkRead)
		if err != nil {
			return totalRead, err
		}
		if chunkRead != int(envelope[1]) {
			return totalRead, asn1error.NewUnexpectedError[int](int(envelope[1]), int(totalRead), "frame truncated").WithUnits("byte(s)")
		}
		return totalRead, nil
	}
	byteCount := envelope[1] & 0x7F
	if byteCount > 6 {
		return totalRead, asn1error.NewErrorf("invalid length encoding")
	}
	var lengthBytes [6]byte
	chunkRead, err = r.Read(lengthBytes[:byteCount])
	totalRead += totalRead
	if err != nil {
		return totalRead, err
	}
	if chunkRead != int(byteCount) {
		return totalRead, asn1error.NewUnexpectedError[int](int(byteCount), chunkRead, "envelope(long) truncated").WithUnits("byte(s)")
	}
	length := 0

	for i := 0; i < int(byteCount); i++ {
		length |= int(lengthBytes[int(byteCount)-i]) << (i * 8)
	}
	v.Bytes = make([]byte, length)
	chunkRead, err = r.Read(v.Bytes)
	totalRead += int64(chunkRead)
	if err != nil {
		return totalRead, err
	}
	if chunkRead != length {
		return totalRead, asn1error.NewUnexpectedError[int](length, chunkRead, "frame(long) truncated").WithUnits("byte(s)")
	}
	return totalRead, nil
}

var _ Packer = &Value{}
var _ Unpacker = &Value{}

func (v *Value) PackAsn1(params *Parameters) (Envelope, []byte, error) {
	err := params.Validate(&v.Envelope)
	if err != nil {
		return Envelope{}, nil, err
	}
	return v.Envelope, v.Bytes, nil
}

func (v *Value) UnpackAsn1(envelope Envelope, bytes []byte) error {
	v.Envelope = envelope
	v.Bytes = bytes
	return nil
}
func (value *Value) UnpackIntoGo(i any) error {
	return value.UnpackIntoGoWithParameters(i, nil)
}
func (value *Value) UnpackIntoGoWithParameters(i any, params *Parameters) error {
	err := params.Validate(&value.Envelope)
	if err != nil {
		return err
	}
	unpacker, err := GetUnpackerFor(i)
	if err != nil {
		return err
	}
	err = unpacker.UnpackAsn1(value.Envelope, value.Bytes)
	if err != nil {
		return err
	}
	return nil
}

func (value *Value) PackFromGo(i any) error {
	return value.PackFromGoWithParameters(i, nil)
}
func (value *Value) PackFromGoWithParameters(i any, params *Parameters) error {
	packer, err := GetPackerFor(i)
	if err != nil {
		return err
	}
	value.Envelope, value.Bytes, err = packer.PackAsn1(params)
	if err != nil {
		return err
	}
	if params != nil {
		err = params.Update(&value.Envelope)
		if err != nil {
			return err
		}
	}
	return nil
}
