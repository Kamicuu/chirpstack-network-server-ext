package storage

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/brocaar/lorawan"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/band"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/logging"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type DeviceExtraConfigurations struct {
	DevEUI          lorawan.EUI64 `db:"dev_eui"`
	EnabledChannels []int32       `db:"enabled_channels"`
}

const (
	ExtraConfigurationKeyTempl = "lora:ns:ec:%s"
)

// Gets channels that are available for given device - on this channels device will send data.
func GetAvailableChannels(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) ([]int32, error) {
	var res []int32

	err := db.QueryRowx(`
		select enabled_channels from device_extra_configs
			where
		dev_eui = $1`,
		devEUI[:],
	).Scan(
		pq.Array(&res),
	)

	if err != nil {
		return res, handlePSQLError(err, "select error")
	}

	return res, nil
}

// Sets channels that are available for given device, other channels will be disabled.
// Update is preformed also in reddis db
func SetAvailableChannels(ctx context.Context, db sqlx.Execer, devEUI lorawan.EUI64, channels []int32) error {
	res, err := db.Exec(`
		update device_extra_configs set 
			enabled_channels = $2
		where
			dev_eui = $1`,
		devEUI[:],
		pq.Array(channels),
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

	err := db.QueryRowx(`
		select * 
		from device_extra_configs 
		where dev_eui = $1`,
		devEUI[:],
	).Scan(
		&c.DevEUI,
		pq.Array(&c.EnabledChannels),
	)

	if err != nil {
		return c, handlePSQLError(err, "select error")
	}

	return c, nil
}

// Create Extra config caches the given device in Redis only.
// Function can also update current existing configurations in redis.
func SetDeviceExtraConfigurationsCache(ctx context.Context, extraConfig DeviceExtraConfigurations) error {
	key := GetRedisKey(ExtraConfigurationKeyTempl, extraConfig.DevEUI)

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

// Gets extra device configs caches the given device in Redis.
func GetDeviceExtraConfigurationsCache(ctx context.Context, devEUI lorawan.EUI64) (DeviceExtraConfigurations, error) {
	var extraConfig DeviceExtraConfigurations
	key := GetRedisKey(ExtraConfigurationKeyTempl, devEUI)

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

// Gets extra device configs  for given device in Redis.
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

// Function creates default configuration for newly added device
func CreateDefaultConfigForDevice(ctx context.Context, db sqlx.Execer, devEUI lorawan.EUI64) error {
	var extraConfig DeviceExtraConfigurations
	extraConfig.DevEUI = devEUI

	//Adding channels
	indices := append(band.Band().GetStandardUplinkChannelIndices(), band.Band().GetCustomUplinkChannelIndices()...)

	channels := make([]int32, len(indices))
	for i, val := range indices {
		channels[i] = int32(val)
	}

	extraConfig.EnabledChannels = channels

	//Saving
	_, err := db.Exec(`
		insert into device_extra_configs (
			dev_eui,
			enabled_channels
		) values ($1, $2)`,
		extraConfig.DevEUI, pq.Array(extraConfig.EnabledChannels),
	)

	if err != nil {
		return handlePSQLError(err, "insert error")
	}

	SetDeviceExtraConfigurationsCache(ctx, extraConfig)

	return nil
}
