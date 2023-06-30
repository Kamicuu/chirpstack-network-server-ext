package storage

import (
	"context"

	"github.com/brocaar/lorawan"
	"github.com/jmoiron/sqlx"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/logging"
	log "github.com/sirupsen/logrus"
)

type DeviceExtraConfigurations struct {
	DevEUI          lorawan.EUI64 `db:"dev_eui"`
	EnabledChannels []uint32      `db:"enabled_channels"`
}

func GetAvailableChannels(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64, channels []uint32) ([]uint32, error) {
	var res []uint32

	err := sqlx.Get(db, &res,
		`select enabled_channels from device_extra_configs
			where
		dev_eui = $1`,
		devEUI[:])

	if err != nil {
		return res, handlePSQLError(err, "select error")
	}

	return res, nil
}

func SetAvailableChannels(ctx context.Context, db sqlx.Execer, devEUI lorawan.EUI64, channels []uint32) error {
	res, err := db.Exec(`
		update device_extra_configs set 
			enabled_channels = $2
		where
			dev_eui = $1`,
		devEUI[:],
		channels,
	)

	if err != nil {
		return handlePSQLError(err, "update error")
	}

	ra, err := res.RowsAffected()
	if err != nil {
		return handlePSQLError(err, "get rows affected error")
	}
	if ra == 0 {
		return ErrDoesNotExist
	}

	log.WithFields(log.Fields{
		"dev_eui":   devEUI,
		"channels:": channels[:],
		"ctx_id":    ctx.Value(logging.ContextIDKey),
	}).Info("device extra config updated - channels set")
	return nil
}

func GetDeviceExtraConfigurations(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (DeviceExtraConfigurations, error) {
	var c DeviceExtraConfigurations

	err := sqlx.Get(db, &c, "select * from device_extra_configs where dev_eui = $1", devEUI[:])
	if err != nil {
		return c, handlePSQLError(err, "select error")
	}

	return c, nil
}
