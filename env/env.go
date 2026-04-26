package env

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type parsedEnvTag struct {
	envKey       string
	defaultValue string
	value        string
}

func parseTag(tag string) *parsedEnvTag {

	var t parsedEnvTag

	for i, s := range strings.Split(tag, ";") {
		if i == 0 {
			// 0 should always be the env key
			t.envKey = s
			t.value = os.Getenv(t.envKey)
		} else {
			values := strings.Split(s, ":")
			switch values[0] {
			case "default":
				t.defaultValue = values[1]
			}
		}

	}

	return &t
}

func parseStruct(dest any) error {
	dstValue := reflect.ValueOf(dest)

	if dstValue.Kind() == reflect.Pointer {

		dstValue = dstValue.Elem()

	}

	dstType := dstValue.Type()

	// iterate through the
	for i := range dstType.NumField() {
		field := dstType.Field(i)
		fieldType := field.Type

		if field.Type.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		switch fieldType.Kind() {
		case reflect.Struct:
			// parse struct
			dvf := dstValue.Field(i)
			if field.Type.Kind() == reflect.Pointer {
				if dvf.IsNil() {
					dvf.Set(reflect.New(fieldType))
				}
				err := parseStruct(dvf.Interface())
				if err != nil {
					return err
				}
			} else {
				err := parseStruct(dvf.Addr().Interface())
				if err != nil {
					return err
				}
			}
		default:
			// parse tag
			envTag, present := field.Tag.Lookup("env")

			if !present {
				continue // ignore
			}

			parsedEnvTag := parseTag(envTag)

			val := parsedEnvTag.value

			if val == "" {
				val = parsedEnvTag.defaultValue
			}

			dvf := dstValue.Field(i)
			if !dvf.CanSet() {
				return fmt.Errorf("cannot set field on %s (kind %s) at index %d", dstType.Name(), dstType.Kind(), i)
			}

			dvf.Set(reflect.ValueOf(val))
		}
	}

	return nil
}

func Parse(dest any, strict bool) error {
	return parseStruct(dest)
}
