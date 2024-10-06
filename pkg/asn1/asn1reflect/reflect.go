package asn1reflect

import (
	"reflect"
	"sync"
	"time"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

var reflectedPackableType = reflect.TypeFor[asn1binary.Packer]()
var reflectedUnpackableType = reflect.TypeFor[asn1binary.Unpacker]()

type reflectedTransformer struct {
	reflectedValue reflect.Value
	reflectHandler reflectHandler
}

var _ asn1binary.Transformer = (*reflectedTransformer)(nil)

func (t *reflectedTransformer) PackAsn1(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
	envelope, bytes, err := t.reflectHandler.PackAsn1(&t.reflectedValue, params)
	if err != nil {
		return asn1binary.Envelope{}, nil, err
	}
	return envelope, bytes, nil
}
func (t *reflectedTransformer) UnpackAsn1(envelope asn1binary.Envelope, bytes []byte) error {
	err := t.reflectHandler.UnpackAsn1(&t.reflectedValue, envelope, bytes)
	return err
}

func getHandlerFor(rType reflect.Type) (reflectHandler, error) {
	lock.Lock()
	defer lock.Unlock()

	if handler, ok := handlerTypeCache[rType]; ok {
		return handler, nil
	}

	kind := rType.Kind()
	if kind <= reflect.Invalid || kind >= reflect.UnsafePointer {
		return nil, asn1error.NewUnimplementedError("unsupported %s", kind.String())
	}
	handler := mapReflectHandler[kind]
	if handler == nil {
		switch kind {
		case reflect.Struct:
			handler = newStructFieldHandler(rType)
		case reflect.Slice:
			handler = newSliceReflectHandler(rType)
		default:
			return nil, asn1error.NewUnimplementedError("unsupported type %s", rType.String())
		}
	}
	handlerTypeCache[rType] = handler
	return handler, nil
}

func getPackerForReflectedValue(reflectedValue reflect.Value) (asn1binary.Packer, error) {
	if reflectedValue.Type().Implements(reflectedPackableType) {
		packable := reflectedValue.Interface().(asn1binary.Packer)
		return packable, nil
	}
	if reflectedValue.Type().Kind() != reflect.Ptr {
		if reflectedValue.CanAddr() {
			ptr := reflectedValue.Addr()
			packable, ok := ptr.Interface().(asn1binary.Packer)
			if ok {
				return packable, nil
			}
		} else {
			ptr := reflect.New(reflectedValue.Type())
			ptr.Elem().Set(reflectedValue)
			packable, ok := ptr.Interface().(asn1binary.Packer)
			if ok {
				return packable, nil
			}
		}
	}
	for reflectedValue.Kind() == reflect.Ptr {
		if reflectedValue.IsNil() {
			return nil, asn1error.NewUnimplementedError("cannot pack a NULL value for %s", reflectedValue.Type().String()).TODO()
		}
		reflectedValue = reflectedValue.Elem()
	}
	handler, err := getHandlerFor(reflectedValue.Type())
	if err != nil {
		return nil, err
	}
	return asn1binary.PackerFunc(func(params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error) {
		return handler.PackAsn1(&reflectedValue, params)
	}), nil
}

func getUnpackerForReflectedValue(reflectedValue reflect.Value) (asn1binary.Unpacker, error) {
	if reflectedValue.Type().Implements(reflectedUnpackableType) {
		if reflectedValue.Kind() == reflect.Ptr && reflectedValue.IsNil() {
			ptr := reflect.New(reflectedValue.Type().Elem())
			reflectedValue.Set(ptr)
		}
		unpackable := reflectedValue.Interface().(asn1binary.Unpacker)
		return unpackable, nil
	}
	if reflectedValue.CanAddr() {
		unpackable, ok := reflectedValue.Addr().Interface().(asn1binary.Unpacker)
		if ok {
			return unpackable, nil
		}
	}
	for reflectedValue.Kind() == reflect.Ptr {
		if reflectedValue.IsNil() {
			ptr := reflect.New(reflectedValue.Type().Elem())
			reflectedValue.Set(ptr)
		}
		reflectedValue = reflectedValue.Elem()
	}
	if !reflectedValue.CanSet() {
		return nil, asn1error.NewErrorf("cannot unpack into a non-settable value - %s", reflectedValue.Type().String()).WithType(asn1error.StructuralError)
	}
	handler, err := getHandlerFor(reflectedValue.Type())
	if err != nil {
		return nil, err
	}
	return asn1binary.UnpackerFunc(func(envelope asn1binary.Envelope, bytes []byte) error {
		return handler.UnpackAsn1(&reflectedValue, envelope, bytes)
	}), nil
}

type reflectHandler interface {
	PackAsn1(reflectedValue *reflect.Value, params *asn1binary.Parameters) (asn1binary.Envelope, []byte, error)
	UnpackAsn1(reflectedValue *reflect.Value, envelope asn1binary.Envelope, bytes []byte) error
}

var lock sync.RWMutex
var handlerTypeCache map[reflect.Type]reflectHandler
var mapReflectHandler [32]reflectHandler

func RegisterReflectKindHandler(kind reflect.Kind, handler reflectHandler) {
	lock.Lock()
	defer lock.Unlock()
	mapReflectHandler[kind] = handler
}

func init() {
	mapReflectHandler[reflect.Bool] = &booleanReflectHandler{}
	mapReflectHandler[reflect.Int] = &integerReflectHandler{}
	mapReflectHandler[reflect.Int8] = &integerReflectHandler{}
	mapReflectHandler[reflect.Int16] = &integerReflectHandler{}
	mapReflectHandler[reflect.Int32] = &integerReflectHandler{}
	mapReflectHandler[reflect.Int64] = &integerReflectHandler{}
	mapReflectHandler[reflect.String] = &stringReflectHandler{}
	mapReflectHandler[reflect.Interface] = &anyReflectHandler{}

	lock.Lock()
	defer lock.Unlock()
	handlerTypeCache = make(map[reflect.Type]reflectHandler)
	handlerTypeCache[reflect.TypeFor[time.Time]()] = &timeReflectHandler{}
}

func getPackerFor(i any) (asn1binary.Packer, error) {
	reflectedValue := reflect.ValueOf(i)
	return getPackerForReflectedValue(reflectedValue)
}

func getUnpackerFor(i any) (asn1binary.Unpacker, error) {
	reflectedValue := reflect.ValueOf(i)
	return getUnpackerForReflectedValue(reflectedValue)
}

func Register() error {
	return asn1binary.RegisterProviderFuncs(11, "reflect", getPackerFor, getUnpackerFor)
}
