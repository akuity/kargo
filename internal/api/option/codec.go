package option

import (
	"bytes"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/cosmos/gogoproto/jsonpb"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	_ connect.Codec = &jsonCodec{}
)

type jsonCodec struct {
	name            string
	m               *jsonpb.Marshaler
	gogoUnmarshaler *jsonpb.Unmarshaler
	unmarshaler     *protojson.UnmarshalOptions
}

func newJSONCodec(name string) connect.Codec {
	return &jsonCodec{
		name: name,
		m: &jsonpb.Marshaler{
			EmitDefaults: true,
		},
		gogoUnmarshaler: &jsonpb.Unmarshaler{},
		unmarshaler:     &protojson.UnmarshalOptions{},
	}
}

func (c *jsonCodec) Name() string {
	return c.name
}

func (c *jsonCodec) Marshal(msg any) ([]byte, error) {
	m, ok := msg.(gogoproto.Message)
	if !ok {
		return nil, errNotProto(msg)
	}
	var b bytes.Buffer
	err := c.m.Marshal(&b, m)
	return b.Bytes(), err
}

func (c *jsonCodec) Unmarshal(data []byte, msg any) error {
	gpm, ok := msg.(gogoproto.Message)
	if !ok {
		return errNotProto(msg)
	}
	if err := c.gogoUnmarshaler.Unmarshal(bytes.NewReader(data), gpm); err == nil {
		return nil
	}
	pm, ok := msg.(proto.Message)
	if !ok {
		return errNotProto(msg)
	}
	return c.unmarshaler.Unmarshal(data, pm)
}

func errNotProto(msg any) error {
	return fmt.Errorf("%T doesn't implement proto.Message", msg)
}
