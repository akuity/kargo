package option

import (
	"bytes"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gogo/protobuf/proto"

	"github.com/akuity/kargo/internal/proto/jsonpb"
)

var (
	_ connect.Codec = &jsonCodec{}
)

type jsonCodec struct {
	name string
	m    *jsonpb.Marshaler
	um   *jsonpb.Unmarshaler
}

func newJSONCodec(name string) connect.Codec {
	return &jsonCodec{
		name: name,
		m:    &jsonpb.Marshaler{},
		um: &jsonpb.Unmarshaler{
			AllowUnknownFields: true,
		},
	}
}

func (c *jsonCodec) Name() string {
	return c.name
}

func (c *jsonCodec) Marshal(msg any) ([]byte, error) {
	pb, ok := msg.(proto.Message)
	if !ok {
		return nil, errNotProto(msg)
	}

	var buf bytes.Buffer
	if err := c.m.Marshal(&buf, pb); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *jsonCodec) Unmarshal(data []byte, msg any) error {
	pb, ok := msg.(proto.Message)
	if !ok {
		return errNotProto(msg)
	}
	return c.um.Unmarshal(bytes.NewReader(data), pb)
}

func errNotProto(msg any) error {
	return fmt.Errorf("%T doesn't implement proto.Message", msg)
}
