package structparser

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// parseArg processes CLI arguments based on the "flag" tag.
func parseArg(flagKey string, fieldVal *reflect.Value, fieldType *reflect.StructField) error {
	argValue, exists := lookupArg(flagKey)
	if !exists || argValue == "" {
		return nil // Skip if the CLI argument wasn't provided
	}

	if !fieldVal.CanSet() {
		return fmt.Errorf("field %s is not public", fieldType.Name)
	}

	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(argValue)

	case reflect.Bool:
		b, err := strconv.ParseBool(argValue)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse arg %q as bool: %w", fieldType.Name, argValue, err)
		}
		fieldVal.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldType.Type.String() == "time.Duration" {
			d, err := time.ParseDuration(argValue)
			if err != nil {
				return fmt.Errorf("field %s: cannot parse arg %q as time.Duration: %w", fieldType.Name, argValue, err)
			}
			fieldVal.SetInt(int64(d))
			break
		}

		parsedInt, err := strconv.ParseInt(argValue, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse arg %q as int: %w", fieldType.Name, argValue, err)
		}
		if fieldVal.OverflowInt(parsedInt) {
			return fmt.Errorf("field %s: arg value %s overflows %s", fieldType.Name, argValue, fieldVal.Kind())
		}
		fieldVal.SetInt(parsedInt)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsedUint, err := strconv.ParseUint(argValue, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse arg %q as uint: %w", fieldType.Name, argValue, err)
		}
		if fieldVal.OverflowUint(parsedUint) {
			return fmt.Errorf("field %s: arg value %s overflows %s", fieldType.Name, argValue, fieldVal.Kind())
		}
		fieldVal.SetUint(parsedUint)

	case reflect.Float32, reflect.Float64:
		parsedFloat, err := strconv.ParseFloat(argValue, 64)
		if err != nil {
			return fmt.Errorf("field %s: cannot parse arg %q as float: %w", fieldType.Name, argValue, err)
		}
		if fieldVal.OverflowFloat(parsedFloat) {
			return fmt.Errorf("field %s: arg value %s overflows %s", fieldType.Name, argValue, fieldVal.Kind())
		}
		fieldVal.SetFloat(parsedFloat)

	default:
		return fmt.Errorf("field %s has unsupported type %s for arg values", fieldType.Name, fieldVal.Kind())
	}

	return nil
}

// lookupArg manually scans os.Args for a specific flag safely.
func lookupArg(flagName string) (string, bool) {
	prefix1 := "--" + flagName
	prefix2 := "-" + flagName

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Match: --flag=value or -flag=value
		if strings.HasPrefix(arg, prefix1+"=") || strings.HasPrefix(arg, prefix2+"=") {
			parts := strings.SplitN(arg, "=", 2)
			return parts[1], true
		}

		// Match: --flag value or -flag value
		if arg == prefix1 || arg == prefix2 {
			// Check if the next argument is the value (and not another flag)
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				return os.Args[i+1], true
			}
			// If no value follows, assume it's a boolean flag (e.g., --verbose)
			return "true", true
		}
	}
	return "", false
}
