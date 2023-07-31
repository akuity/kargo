package option

import (
	"fmt"

	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	_ connect.Codec = &jsonCodec{}
)

type jsonCodec struct {
	name string
	m    *protojson.MarshalOptions
	um   *protojson.UnmarshalOptions
}

func newJSONCodec(name string) connect.Codec {
	return &jsonCodec{
		name: name,
		m: &protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true,
		},
		um: &protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
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
	return c.m.Marshal(m)
}

func (c *jsonCodec) Unmarshal(data []byte, msg any) error {
	m, ok := msg.(proto.Message)
	if !ok {
		return errNotProto(msg)
	}
	return c.um.Unmarshal(data, m)
}

func errNotProto(msg any) error {
	return fmt.Errorf("%T doesn't implement proto.Message", msg)
}
