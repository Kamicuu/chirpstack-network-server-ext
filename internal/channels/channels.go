package channels

import (
	"github.com/brocaar/lorawan"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/band"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/storage"
)

// HandleChannelReconfigure handles the reconfiguration of active channels
// on the node. This is needed in case only a sub-set of channels is used
// (e.g. for the US band) or when a reconfiguration of active channels
// happens.
func HandleChannelReconfigure(ds storage.DeviceSession) ([]storage.MACCommandBlock, error) {

	// quite ugly workaround for ability to disable default channels
	// if any of default channels is not present in EnabledUplinkChannels, then
	// GetLinkADRReqPayloadsForEnabledUplinkChannelIndices creates ADR paylode that does
	// not allow to disable default channels, so we do not get paylode from this method
	// if any default channel is not present
	var defChan int
	for _, i := range ds.EnabledUplinkChannels {
		if i == 0 || i == 1 || i == 2 {
			defChan++
		}
	}

	if defChan != 3 {
		return nil, nil
	}

	//end workaround

	payloads := band.Band().GetLinkADRReqPayloadsForEnabledUplinkChannelIndices(ds.EnabledUplinkChannels)
	if len(payloads) == 0 {
		return nil, nil
	}

	payloads[len(payloads)-1].TXPower = uint8(ds.TXPowerIndex)
	payloads[len(payloads)-1].DataRate = uint8(ds.DR)
	payloads[len(payloads)-1].Redundancy.NbRep = ds.NbTrans

	block := storage.MACCommandBlock{
		CID: lorawan.LinkADRReq,
	}
	for i := range payloads {
		block.MACCommands = append(block.MACCommands, lorawan.MACCommand{
			CID:     lorawan.LinkADRReq,
			Payload: &payloads[i],
		})
	}

	return []storage.MACCommandBlock{block}, nil
}
