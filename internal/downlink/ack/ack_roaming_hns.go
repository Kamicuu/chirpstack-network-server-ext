package ack

import (
	"context"

	"github.com/kamicuu/chirpstack-api/go/v3/gw"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/storage"
)

// HandleDownlinkXmitDataAns handles an ack as hNS.
func HandleRoamingTxAck(ctx context.Context, txAck gw.DownlinkTXAck) error {
	actx := ackContext{
		ctx:                 ctx,
		DB:                  storage.DB(),
		DownlinkTXAck:       &txAck,
		DownlinkTXAckStatus: gw.TxAckStatus_OK,
	}

	for _, t := range handleDownlinkTXAckTasks {
		if err := t(&actx); err != nil {
			return err
		}
	}

	return nil
}
