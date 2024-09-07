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

func Variations(keys ...string) []string {
	var keysToTry []string

	for _, key := range keys {
		keysToTry = append(keysToTry, key)
		if strings.HasSuffix(key, "s") {
			key_without_s := key[:len(key)-1]
			keysToTry = append(keysToTry, key_without_s)
		} else {
			key_with_s := key + "s"
			keysToTry = append(keysToTry, key_with_s)
		}
		if strings.HasSuffix(key, "y") {
			key_without_y := key[:len(key)-1] + "ies"
			keysToTry = append(keysToTry, key_without_y)
		}
	}
	return keysToTry
}

func CheckKeys(args Config, keys ...string) error {
	keyVariations := Variations(keys...)
	for k := range args {
		if !slices.Contains(keyVariations, k) {
			return fmt.Errorf("unexpected key %s", k)
		}
	}
	return nil
}

func GetArg[T any](cfg Config, key string, defaultValue T) (T, error) {

	var keysToTry []string = []string{key}
	if strings.HasSuffix(key, "s") {
		key_without_s := key[:len(key)-1]
		keysToTry = append(keysToTry, key_without_s)
	} else {
		key_with_s := key + "s"
		keysToTry = append(keysToTry, key_with_s)
	}
	if strings.HasSuffix(key, "y") {
		key_without_y := key[:len(key)-1] + "ies"
		keysToTry = append(keysToTry, key_without_y)
	}

	for _, keyToTry := range keysToTry {
		if v, ok := cfg[keyToTry]; ok {
			tv, ok := v.(T)
			if !ok {
				return defaultValue, fmt.Errorf("invalid type %s for key %s, expected %s", reflect.TypeOf(v), keyToTry, reflect.TypeOf(defaultValue))
			}
			return tv, nil
		}
	}

	return defaultValue, nil
}
