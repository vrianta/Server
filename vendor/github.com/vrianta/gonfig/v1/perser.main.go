package structparser

import (
	"errors"
	"fmt"
	"reflect"
)

func New[T any](crashOnFail bool) T {
	var cfg T

	Parse(&cfg, crashOnFail)

	return cfg
}

/*
Function wchich will get a any struct and loop thorugh it veriables.
look for env tag and populate it
*/
func Parse[T any](ctx *T, crashOnFail bool) (*T, error) {
	// 1. Ensure we passed a pointer, otherwise we cannot mutate the struct
	val := reflect.ValueOf(ctx)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil, errors.New("must pass a non-nil pointer to a struct")
	}

	if err := perserValue(val, crashOnFail); err != nil {
		return nil, err
	}

	return ctx, nil
}

func perserValue(val reflect.Value, crashOnFail bool) error {
	// 2. Get the underlying struct value and type
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.New("provided value is not a pointer to a struct")
	}

	typ := val.Type()

	// 3. Loop through all fields of the struct
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		if fieldVal.Kind() == reflect.Struct {
			if err := perserValue(fieldVal, crashOnFail); err != nil {
				return err
			}
			continue
		}

		if !fieldVal.CanSet() {
			return fmt.Errorf("field %s is not public", fieldType.Name)
		}

		if envKey, hasTag := fieldType.Tag.Lookup("env"); hasTag && envKey != "" {
			if err := parseEnv(envKey, &fieldVal, &fieldType); err == nil {
				continue
			}
		}

		if argName, hasTag := fieldType.Tag.Lookup("arg"); hasTag && argName != "" {
			if err := parseArg(argName, &fieldVal, &fieldType); err == nil {
				continue
			}
		}

		if def, hasTag := fieldType.Tag.Lookup("default"); hasTag && def != "" {
			if err := parseDefault(def, &fieldVal, &fieldType); err == nil {
				continue
			}
		}

		// 3. Process Required Validation Last
		if _, isRequired := fieldType.Tag.Lookup("required"); isRequired {
			// If the tag is present, check if the field was left at its zero-value
			if fieldVal.IsZero() {
				if crashOnFail {
					panic("field " + fieldType.Name + "is required but has no value")
				}
				return fmt.Errorf("field %s is required but has no value", fieldType.Name)
			}
		}

	}

	return nil
}
