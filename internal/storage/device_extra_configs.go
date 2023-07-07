package storage

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/brocaar/lorawan"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/logging"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type DeviceExtraConfigurations struct {
	DevEUI          lorawan.EUI64 `db:"dev_eui"`
	EnabledChannels []uint32      `db:"enabled_channels"`
}

const (
	ExtraConfigurationKeyTempl = "lora:ns:ec:%s"
)

// Gets channels that are available for given device - on this channels device will send data.
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

// Sets channels that are available for given device, other channels will be disabled.
// Update is preformed also in reddis db
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
		val, err := GetDeviceExtraConfigurationsCache(ctx, devEUI)

		if err != nil {
			return handlePSQLError(err, "update error, cache not updated")
		}

		val.EnabledChannels = channels
		SetDeviceExtraConfigurationsCache(ctx, val)

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

// Gets Extra config options for devie from Postgress DB
func GetDeviceExtraConfigurations(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (DeviceExtraConfigurations, error) {
	var c DeviceExtraConfigurations

	err := sqlx.Get(db, &c, "select * from device_extra_configs where dev_eui = $1", devEUI[:])
	if err != nil {
		return c, handlePSQLError(err, "select error")
	}

	return c, nil
}

// CreateDeviceProfileCache caches the given device in Redis.
// Function can also update current existing configurations.
func SetDeviceExtraConfigurationsCache(ctx context.Context, extraConfig DeviceExtraConfigurations) error {
	key := GetRedisKey(DeviceProfileKeyTempl, extraConfig.DevEUI)

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(extraConfig); err != nil {
		return errors.Wrap(err, "gob encode extra-config error")
	}

	err := RedisClient().Set(ctx, key, buf.Bytes(), deviceSessionTTL).Err()
	if err != nil {
		return errors.Wrap(err, "set device-profile error")
	}

	return nil
}

// CreateDeviceProfileCache caches the given device in Redis.
func GetDeviceExtraConfigurationsCache(ctx context.Context, devEUI lorawan.EUI64) (DeviceExtraConfigurations, error) {
	var extraConfig DeviceExtraConfigurations
	key := GetRedisKey(DeviceProfileKeyTempl, devEUI)

	val, err := RedisClient().Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return extraConfig, ErrDoesNotExist
		}
		return extraConfig, errors.Wrap(err, "get error")
	}

	err = gob.NewDecoder(bytes.NewReader(val)).Decode(&extraConfig)
	if err != nil {
		return extraConfig, errors.Wrap(err, "gob decode error")
	}

	return extraConfig, nil
}

// Gets CreateDeviceProfileCache for given device in Redis.
// If device not exists in cache, function will get config from DB and save to cache
func GetAndCacheDeviceExtraConfigurationsCache(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (DeviceExtraConfigurations, error) {
	extraConfig, err := GetDeviceExtraConfigurationsCache(ctx, devEUI)
	if err == nil {
		return extraConfig, nil
	}

	if err != ErrDoesNotExist {
		log.WithFields(log.Fields{
			"devEUI": devEUI,
		}).WithError(err).Error("get extra config cache error")
		// we don't return as we can still fall-back onto db retrieval

		extraConfig, err = GetDeviceExtraConfigurations(ctx, db, devEUI)
		if err != nil {
			return DeviceExtraConfigurations{}, errors.Wrap(err, "get extra config error")
		}
	}

	err = SetDeviceExtraConfigurationsCache(ctx, extraConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"ctx_id": ctx.Value(logging.ContextIDKey),
			"devEUI": devEUI,
		}).WithError(err).Error("create extra config cache error")
	}

	return extraConfig, nil
}
