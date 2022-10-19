package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func output(obj any, out io.Writer, format string) error {
	var bytes []byte
	var err error
	switch strings.ToLower(format) {
	case flagOutputJSON:
		bytes, err = json.MarshalIndent(obj, "", "  ")
	case flagOutputYAML:
		bytes, err = yaml.Marshal(obj)
	default:
		return errors.Errorf("unsupported output format %q", format)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(bytes))
	return nil
}
