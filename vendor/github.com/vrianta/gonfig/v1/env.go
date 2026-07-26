package structparser

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

/*
Perse only Environment tag which is env
*/
func parseEnv(envKey string, fieldVal *reflect.Value, fieldType *reflect.StructField) error {
	envValue, exists := os.LookupEnv(envKey)
	if !exists || envValue == "" {
		return fmt.Errorf("Environment Value is Empty") // Skip if the env variable isn't set
	}

	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(envValue)

	case reflect.Bool:
		// strconv.ParseBool automatically handles true, false, 1, 0, T, F, True, False
		b, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse %q as bool: %w", fieldType.Name, envValue, err)
		}
		fieldVal.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special check: handle time.Duration if your community wants to support it
		if fieldType.Type.String() == "time.Duration" {
			d, err := time.ParseDuration(envValue)
			if err != nil {
				return fmt.Errorf("field %s: cannot parse %q as time.Duration: %w", fieldType.Name, envValue, err)
			}
			fieldVal.SetInt(int64(d))
			break
		}

		parsedInt, err := strconv.ParseInt(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse %q as int: %w", fieldType.Name, envValue, err)
		}
		if fieldVal.OverflowInt(parsedInt) {
			return fmt.Errorf("field %s: value %s overflows %s", fieldType.Name, envValue, fieldVal.Kind())
		}
		fieldVal.SetInt(parsedInt)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsedUint, err := strconv.ParseUint(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse %q as uint: %w", fieldType.Name, envValue, err)
		}
		if fieldVal.OverflowUint(parsedUint) {
			return fmt.Errorf("field %s: value %s overflows %s", fieldType.Name, envValue, fieldVal.Kind())
		}
		fieldVal.SetUint(parsedUint)

	case reflect.Float32, reflect.Float64:
		parsedFloat, err := strconv.ParseFloat(envValue, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse %q as float: %w", fieldType.Name, envValue, err)
		}
		if fieldVal.OverflowFloat(parsedFloat) {
			return fmt.Errorf("field %s: value %s overflows %s", fieldType.Name, envValue, fieldVal.Kind())
		}
		fieldVal.SetFloat(parsedFloat)

	default:
		return fmt.Errorf("field %s has unsupported type %s", fieldType.Name, fieldVal.Kind())
	}
	return nil

}
