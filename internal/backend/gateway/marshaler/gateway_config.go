package marshaler

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/kamicuu/chirpstack-api/go/v3/gw"
)

// MarshalGatewayConfiguration marshals the GatewayConfiguration.
func MarshalGatewayConfiguration(t Type, gc gw.GatewayConfiguration) ([]byte, error) {
	var b []byte
	var err error

	switch t {
	case Protobuf:
		b, err = proto.Marshal(&gc)
	case JSON:
		var str string
		m := &jsonpb.Marshaler{
			EmitDefaults: true,
		}
		str, err = m.MarshalToString(&gc)
		b = []byte(str)
	}

	return b, err
}
