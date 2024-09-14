package asn1reflect

import (
	"bytes"
	"reflect"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

// --------------------------------------------
type sliceReflectHandler struct {
}

func (srh *sliceReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {

	len := reflectedValue.Len()

	b := bytes.Buffer{}
	var asn1Value asn1binary.Value
	for i := 0; i < len; i++ {
		elem := reflectedValue.Index(i)
		packer, err := getPackerForReflectedValue(elem)
		if err != nil {
			return asn1binary.Envelope{}, nil, err
		}
		asn1Value.Envelope, asn1Value.Bytes, err = packer.PackAsn1(params)
		if err != nil {
			return asn1binary.Envelope{}, nil, err
		}
		elemChunk, err := asn1Value.Marshal()
		if err != nil {
			return asn1binary.Envelope{}, nil, err
		}
		b.Write(elemChunk)
	}
	return asn1binary.Envelope{Tag: asn1core.TagSequence}, b.Bytes(), nil
}

const maxint = int(^uint(0) >> 1)

func (srh *sliceReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {

	maxCount := maxint
	index := 0
	isSlice := false
	switch reflectedValue.Kind() {
	case reflect.Slice:
		isSlice = true
	case reflect.Array:
		maxCount = reflectedValue.Len()
	default:
		return asn1core.NewUnexpectedError(reflect.Slice, reflectedValue.Kind(), "unexpected kind")
	}

	var asn1Value asn1binary.Value
	for len(bytes) > 0 {
		if index >= maxCount {
			return asn1core.NewUnexpectedError(maxCount, index+1, "too many elements")
		}
		tail, err := asn1Value.Unmarshal(bytes)
		if err != nil {
			return err
		}
		elem := reflect.New(reflectedValue.Type().Elem()).Elem()
		unpacker, err := getUnpackerForReflectedValue(elem)
		if err != nil {
			return err
		}
		err = unpacker.UnpackAsn1(asn1Value.Envelope, asn1Value.Bytes)
		if err != nil {
			return err
		}
		if isSlice {
			updatedSlice := reflect.Append(*reflectedValue, elem)
			reflectedValue.Set(updatedSlice)
		} else { //isArray
			reflectedValue.Index(index).Set(elem)
		}
		bytes = tail
		index++
	}
	if !isSlice && index < maxCount {
		return asn1core.NewUnexpectedError(maxCount, index, "too few elements")
	}

	return nil
}

// --------------------------------------------

type byteSliceReflectHandler struct {
}

func (s *byteSliceReflectHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	return asn1binary.Envelope{}, nil, asn1core.NewUnimplementedError("byteSliceReflectHandler.PackAsn1")
}
func (s *byteSliceReflectHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	if envelope.Tag != asn1core.TagSequence {
		return asn1core.NewUnexpectedError(asn1core.TagOctetString, envelope.Tag, "unexpected tag")
	}
	return asn1core.NewUnimplementedError("byteSliceReflectHandler.UnpackAsn1")
}

func newSliceReflectHandler(rType reflect.Type) reflectHandler {
	if rType.Elem().Kind() == reflect.Uint8 {
		return &byteSliceReflectHandler{}
	}
	srh := &sliceReflectHandler{}
	return srh
}
