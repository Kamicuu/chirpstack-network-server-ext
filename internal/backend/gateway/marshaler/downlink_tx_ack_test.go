package marshaler

import (
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/kamicuu/chirpstack-api/go/v3/gw"
)

func TestUnmarshalDownlinkTXAck(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		assert := require.New(t)

		in := gw.DownlinkTXAck{
			GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			Token:     12345,
			Error:     "Boom!",
		}
		m := jsonpb.Marshaler{}
		str, err := m.MarshalToString(&in)
		assert.NoError(err)

		var out gw.DownlinkTXAck
		typ, err := UnmarshalDownlinkTXAck([]byte(str), &out)
		assert.NoError(err)
		assert.Equal(JSON, typ)
		assert.True(proto.Equal(&in, &out))
	})

	t.Run("Protobuf", func(t *testing.T) {
		assert := require.New(t)

		in := gw.DownlinkTXAck{
			GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			Token:     12345,
			Error:     "Boom!",
		}
		b, err := proto.Marshal(&in)
		assert.NoError(err)

		var out gw.DownlinkTXAck
		typ, err := UnmarshalDownlinkTXAck(b, &out)
		assert.NoError(err)
		assert.Equal(Protobuf, typ)
		assert.True(proto.Equal(&in, &out))
	})
}
