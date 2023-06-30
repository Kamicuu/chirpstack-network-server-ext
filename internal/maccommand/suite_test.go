package maccommand

import (
	"context"

	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/storage"
	"github.com/kamicuu/chirpstack-network-server-ext/v3/internal/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestBase struct {
	suite.Suite
}

func (ts *TestBase) SetupSuite() {
	assert := require.New(ts.T())
	conf := test.GetConfig()
	assert.NoError(storage.Setup(conf))
}

func (ts *TestBase) SetupTest() {
	storage.RedisClient().FlushAll(context.Background())
}
