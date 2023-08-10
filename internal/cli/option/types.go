package option

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/samber/mo"
	"github.com/spf13/pflag"
)

type Optional[T any] interface {
	pflag.Value

	/* partial methods of mo.Option */

	IsPresent() bool
	Get() (T, bool)
	OrElse(fallback T) T
}

type optionalBool struct {
	mo.Option[bool]
}

func (o *optionalBool) String() string {
	v, ok := o.Get()
	if !ok {
		return ""
	}
	return strconv.FormatBool(v)
}

func (o *optionalBool) Set(value string) error {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return errors.Wrap(err, "parse bool")
	}
	o.Option = mo.Some[bool](v)
	return nil
}

func (o *optionalBool) Type() string {
	return "bool"
}

func OptionalBool() Optional[bool] {
	return &optionalBool{
		Option: mo.None[bool](),
	}
}

type optionalString struct {
	mo.Option[string]
}

func (o *optionalString) String() string {
	v, ok := o.Get()
	if !ok {
		return ""
	}
	return v
}

func (o *optionalString) Set(value string) error {
	o.Option = mo.Some[string](value)
	return nil
}

func (o *optionalString) Type() string {
	return "string"
}

func OptionalString() Optional[string] {
	return &optionalString{
		Option: mo.None[string](),
	}
}
