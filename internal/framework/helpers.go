package framework

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func IsIdentifier(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("empty string")
	}
	if len(s) > 64 {
		return fmt.Errorf("string too long")
	}
	for i, c := range s {
		if i == 0 && !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return fmt.Errorf("first character must be a letter")
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("invalid character %c", c)
		}
	}
	return nil
}

type Config map[string]any

func Variations(fields ...string) []string {
	var fieldsToTry []string

	for _, field := range fields {
		fieldsToTry = append(fieldsToTry, field)
		if strings.HasSuffix(field, "s") {
			field_without_s := field[:len(field)-1]
			fieldsToTry = append(fieldsToTry, field_without_s)
		} else {
			field_with_s := field + "s"
			fieldsToTry = append(fieldsToTry, field_with_s)
		}
		if strings.HasSuffix(field, "y") {
			field_without_y := field[:len(field)-1] + "ies"
			fieldsToTry = append(fieldsToTry, field_without_y)
		}
	}
	return fieldsToTry
}

func CheckFields(args Config, fields ...string) error {
	fieldVariations := Variations(fields...)
	unexpectedfields := make([]string, 0)
	for k := range args {
		if !slices.Contains(fieldVariations, k) {
			unexpectedfields = append(unexpectedfields, fmt.Sprintf("%q", k))
		}
	}
	if len(unexpectedfields) > 0 {
		return fmt.Errorf("unexpected fields: %s", strings.Join(unexpectedfields, ", "))
	}
	return nil
}

// also removes the field from the cfg
func consumeOptionalArg[T any](cfg Config, field string, defaultValue *T) (*T, error) {

	var fieldsToTry []string = []string{field}
	if strings.HasSuffix(field, "s") {
		field_without_s := field[:len(field)-1]
		fieldsToTry = append(fieldsToTry, field_without_s)
	} else {
		field_with_s := field + "s"
		fieldsToTry = append(fieldsToTry, field_with_s)
	}
	if strings.HasSuffix(field, "y") {
		field_without_y := field[:len(field)-1] + "ies"
		fieldsToTry = append(fieldsToTry, field_without_y)
	}

	for _, fieldToTry := range fieldsToTry {
		if v, ok := cfg[fieldToTry]; ok {
			tv, ok := v.(T)
			if !ok {
				requiredType := reflect.TypeOf(defaultValue).Elem()
				gotType := reflect.TypeOf(v)

				switch requiredType.Kind() {
				case reflect.Slice, reflect.Array:
					elemType := requiredType.Elem()
					if elemType == gotType {
						newArray := reflect.MakeSlice(reflect.SliceOf(elemType), 1, 1)
						newArray.Index(0).Set(reflect.ValueOf(v))
						arrayImpl := newArray.Interface().(T)
						delete(cfg, fieldToTry)
						return &arrayImpl, nil
					}
					if gotType.Kind() == reflect.Slice || gotType.Kind() == reflect.Array {
						count := reflect.ValueOf(v).Len()
						newArray := reflect.MakeSlice(reflect.SliceOf(elemType), count, count)
						for i := 0; i < count; i++ {
							elem := reflect.ValueOf(v).Index(i)
							if elem.Kind() == reflect.Interface && elem.NumMethod() == 0 {
								elem = elem.Elem()
							}

							if !elem.CanConvert(elemType) {
								err := fmt.Errorf("invalid type %s in list for field %q, expected %s", elem.Type(), fieldToTry, elemType)
								return defaultValue, err
							}
							elem = elem.Convert(elemType)
							newArray.Index(i).Set(elem)
						}
						arrayImpl := newArray.Interface().(T)
						delete(cfg, fieldToTry)
						return &arrayImpl, nil

					}

				}
				err := fmt.Errorf("invalid type %s for field %q, expected %s", gotType, fieldToTry, requiredType)
				return defaultValue, err
			}
			delete(cfg, fieldToTry)
			return &tv, nil
		}
	}

	return defaultValue, nil
}

func ConsumeOptionalArg[T any](cfg Config, field string, defaultValue T) (T, error) {
	tp, err := consumeOptionalArg(cfg, field, &defaultValue)
	if err != nil {
		return defaultValue, err
	}
	if tp == nil {
		return defaultValue, nil
	}
	return *tp, nil
}

func ConsumeArg[T any](cfg Config, field string) (T, error) {
	tp, err := consumeOptionalArg[T](cfg, field, nil)
	if err != nil {
		var null T
		return null, err
	}
	if tp == nil {
		var null T
		return null, fmt.Errorf("missing required field %q", field)
	}

	return *tp, nil
}
