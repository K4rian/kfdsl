package arguments

import (
	"fmt"

	"github.com/spf13/cast"
)

type ParsableArgument interface {
	Parse() error
	Name() string
	FormattedValue() string
	IsSensitive() bool
}

type ParseFunction[T any] func(a *Argument[T]) (T, error)

type FormatFunction[T any] func(a *Argument[T]) string

type Argument[T any] struct {
	name           string            // Argument name
	rawValue       T                 // Raw input from CLI
	parsedValue    T                 // Parsed value
	formattedValue string            // Formatted value or parsed value
	parseFunc      ParseFunction[T]  // Parser function or raw value
	formatFunc     FormatFunction[T] // Format function or string
	sensitive      bool              // True if the argument contains sensitive data
}

func New[T any](name string, rawValue T, parseFunc ParseFunction[T], formatFunc FormatFunction[T], sensitive bool) *Argument[T] {
	if parseFunc == nil {
		parseFunc = defaultParser[T]
	}
	if formatFunc == nil {
		formatFunc = defaultFormatter[T]
	}
	return &Argument[T]{
		name:       name,
		rawValue:   rawValue,
		parseFunc:  parseFunc,
		formatFunc: formatFunc,
		sensitive:  sensitive,
	}
}

func (a *Argument[T]) Name() string {
	return a.name
}

func (a *Argument[T]) RawValue() T {
	return a.rawValue
}

func (a *Argument[T]) Value() T {
	return a.parsedValue
}

func (a *Argument[T]) FormattedValue() string {
	return a.formattedValue
}

func (a *Argument[T]) IsSensitive() bool {
	return a.sensitive
}

func (a *Argument[T]) String() string {
	return fmt.Sprintf("%v", a.parsedValue)
}

// func (a *Argument[T]) SetValue(v T) {
// 	a.parsedValue = v
// }

// func (a *Argument[T]) SetFormattedValue(v string) {
// 	a.formattedValue = v
// }

func (a *Argument[T]) SetParserFunction(f ParseFunction[T]) {
	a.parseFunc = f
}

func (a *Argument[T]) SetFormatterFunction(f FormatFunction[T]) {
	a.formatFunc = f
}

func (a *Argument[T]) Parse() error {
	pVal, err := a.parseFunc(a)
	if err != nil {
		return err
	}
	a.parsedValue = pVal
	a.formattedValue = a.formatFunc(a)
	return nil
}

func convertType[T any](val any) (T, error) {
	v, ok := val.(T)
	if !ok {
		return v, fmt.Errorf("type assertion failed: expected %T, got %T", v, val)
	}
	return v, nil
}

func defaultParser[T any](a *Argument[T]) (T, error) {
	var err error

	rv := a.rawValue
	switch val := any(rv).(type) {
	case int8, int16, int32, int64:
		i, err := cast.ToInt64E(val)
		if err == nil {
			return convertType[T](i)
		}
	case uint, uint8, uint16, uint32, uint64:
		u, err := cast.ToUint64E(val)
		if err == nil {
			return convertType[T](u)
		}
	case int:
		i, err := cast.ToIntE(val)
		if err == nil {
			return convertType[T](i)
		}
	case float64:
		f, err := cast.ToFloat64E(val)
		if err == nil {
			return convertType[T](f)
		}
	case bool:
		b, err := cast.ToBoolE(val)
		if err == nil {
			return convertType[T](b)
		}
	case string:
		s, err := cast.ToStringE(val)
		if err == nil {
			return convertType[T](s)
		}
	default:
		return *new(T), fmt.Errorf("unsupported type; provide a custom parser or valid type")
	}
	return *new(T), err
}

func defaultFormatter[T any](a *Argument[T]) string {
	return fmt.Sprintf("%v", a.Value())
}
