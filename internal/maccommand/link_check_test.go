package maccommand

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/lorawan"
	"github.com/kamicuu/chirpstack-api/go/v3/gw"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/band"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/helpers"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/models"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/storage"
)

type LinkCheckTestSuite struct {
	TestBase
}

func (ts *LinkCheckTestSuite) TestLinkCheckReq() {
	assert := require.New(ts.T())
	ctx := context.Background()

	ds := storage.DeviceSession{
		DevEUI:                [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		EnabledUplinkChannels: []int{0, 1},
	}
	assert.NoError(storage.SaveDeviceSession(ctx, ds))

	block := storage.MACCommandBlock{
		CID: lorawan.LinkCheckReq,
		MACCommands: storage.MACCommands{
			lorawan.MACCommand{
				CID: lorawan.LinkCheckReq,
			},
		},
	}

	rxPacket := models.RXPacket{
		TXInfo: &gw.UplinkTXInfo{},
		RXInfoSet: []*gw.UplinkRXInfo{
			{
				LoraSnr: 5,
			},
		},
	}

	assert.NoError(helpers.SetUplinkTXInfoDataRate(rxPacket.TXInfo, 2, band.Band()))

	resp, err := Handle(ctx, &ds, storage.DeviceProfile{}, storage.ServiceProfile{}, nil, block, nil, rxPacket)
	assert.NoError(err)

	assert.Len(resp, 1)
	assert.Equal(storage.MACCommandBlock{
		CID: lorawan.LinkCheckAns,
		MACCommands: storage.MACCommands{
			{
				CID: lorawan.LinkCheckAns,
				Payload: &lorawan.LinkCheckAnsPayload{
					GwCnt:  1,
					Margin: 20, // 5 - -15 (see SpreadFactorToRequiredSNRTable)
				},
			},
		},
	}, resp[0])
}

func TestLinkCheck(t *testing.T) {
	suite.Run(t, new(LinkCheckTestSuite))
}
