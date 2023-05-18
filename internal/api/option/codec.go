package option

import (
	"bytes"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/cosmos/gogoproto/proto"
)

var (
	_ connect.Codec = &jsonCodec{}
)

type jsonCodec struct {
	name string
	m    *jsonpb.Marshaler
	u    *jsonpb.Unmarshaler
}

func newJSONCodec(name string) connect.Codec {
	return &jsonCodec{
		name: name,
		m: &jsonpb.Marshaler{
			EmitDefaults: true,
		},
		u: &jsonpb.Unmarshaler{},
	}
}

func (c *jsonCodec) Name() string {
	return c.name
}

func (c *jsonCodec) Marshal(msg any) ([]byte, error) {
	m, ok := msg.(proto.Message)
	if !ok {
		return nil, errNotProto(msg)
	}
	var b bytes.Buffer
	err := c.m.Marshal(&b, m)
	return b.Bytes(), err
}

func (c *jsonCodec) Unmarshal(data []byte, msg any) error {
	m, ok := msg.(proto.Message)
	if !ok {
		return errNotProto(msg)
	}
	return c.u.Unmarshal(bytes.NewReader(data), m)
}

func errNotProto(msg any) error {
	return fmt.Errorf("%T doesn't implement proto.Message", msg)
}
