package asn1reflect

import (
	"bytes"
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type fieldsHelper struct {
	hasEnvelope bool
	fields      []reflect.StructField
	params      []*asn1binary.Parameters
}

var fieldHelperCache map[reflect.Type]*fieldsHelper

var envelopeType = reflect.TypeFor[asn1binary.Envelope]()

func fieldHelperFor(rType reflect.Type) (*fieldsHelper, error) {
	lock.RLock()
	helper, ok := fieldHelperCache[rType]
	lock.RUnlock()
	if ok {
		return helper, nil
	}
	helper = &fieldsHelper{}

	nFields := rType.NumField()
	for i := 0; i < nFields; i++ {
		field := rType.Field(i)
		paramsText := field.Tag.Get("asn1")
		params, err := asn1binary.ParseParameters(paramsText)
		if err != nil {
			return nil, err
		}
		helper.fields = append(helper.fields, field)
		helper.params = append(helper.params, params)
	}

	if len(helper.fields) == 0 {
		return nil, asn1error.NewUnimplementedError("no fields")
	}
	helper.hasEnvelope = (helper.fields[0].Type == envelopeType)

	lock.Lock()
	defer lock.Unlock()
	if fieldHelperCache == nil {
		fieldHelperCache = make(map[reflect.Type]*fieldsHelper)
	}
	fieldHelperCache[rType] = helper
	return helper, nil
}

type structFieldHandler struct {
}

func (sfh *structFieldHandler) PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {

	rType := reflectedValue.Type()
	fieldsHelper, err := fieldHelperFor(rType)
	if err != nil {
		return asn1binary.Envelope{}, nil, asn1error.NewErrorf("preparing to unpack into %s", reflectedValue.Type()).WithCause(err)
	}

	i := 0
	e := asn1binary.Envelope{}
	if fieldsHelper.hasEnvelope {
		i++
		e = reflectedValue.Field(0).Interface().(asn1binary.Envelope)
	}
	err = params.Update(&e)
	if err != nil {
		return asn1binary.Envelope{}, nil, asn1error.NewErrorf("using envelope from struct").WithCause(err)
	}

	b := bytes.Buffer{}
	var asn1Value asn1binary.Value
	for i < len(fieldsHelper.fields) {
		fieldValue := reflectedValue.Field(i)
		fieldParams := fieldsHelper.params[i]
		packer, err := getPackerForReflectedValue(fieldValue)
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1error.NewErrorf("packing field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}

		asn1Value.Envelope, asn1Value.Bytes, err = packer.PackAsn1(fieldParams)
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1error.NewErrorf("packing field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		i++
		elemChunk, err := asn1Value.Marshal()
		if err != nil {
			return asn1binary.Envelope{}, nil, asn1error.NewErrorf("marshalling field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		b.Write(elemChunk)
	}
	return e, b.Bytes(), nil
}
func (sfh *structFieldHandler) UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error {
	rType := reflectedValue.Type()
	fieldsHelper, err := fieldHelperFor(rType)
	if err != nil {
		return asn1error.NewErrorf("preparing to unpack into %s", reflectedValue.Type()).WithCause(err)

	}

	i := 0
	if fieldsHelper.hasEnvelope {
		i++
		reflectedValue.Field(0).Set(reflect.ValueOf(envelope))
		err = fieldsHelper.params[0].Validate(&envelope)
		if err != nil {
			return asn1error.NewErrorf("updating envelope in struct").WithCause(err)
		}
	}

	var asn1Value asn1binary.Value
	for len(bytes) > 0 {
		if i >= len(fieldsHelper.fields) {
			return asn1error.NewUnexpectedError(len(fieldsHelper.fields), i+1, "too many elements")
		}
		tail, err := asn1Value.Unmarshal(bytes)
		if err != nil {
			return asn1error.NewErrorf("unmarshalling field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		fieldParams := fieldsHelper.params[i]
		field := reflectedValue.Field(i)
		unpacker, err := getUnpackerForReflectedValue(field)
		if err != nil {
			return asn1error.NewErrorf("choosing unpacking for field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		err = fieldParams.Validate(&asn1Value.Envelope)
		if err != nil {
			return asn1error.NewErrorf("validating field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		err = unpacker.UnpackAsn1(asn1Value.Envelope, asn1Value.Bytes)
		if err != nil {
			return asn1error.NewErrorf("unpacking field %q", fieldsHelper.fields[i].Name).WithCause(err)
		}
		i++
		bytes = tail
	}
	if i < len(fieldsHelper.fields) {
		return asn1error.NewUnexpectedError(len(fieldsHelper.fields), i, "too few elements")
	}

	return nil
}

func newStructFieldHandler(_ reflect.Type) reflectHandler {
	sfh := &structFieldHandler{}
	return sfh
}
