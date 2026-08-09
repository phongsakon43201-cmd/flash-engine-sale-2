package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientParsesManagedRedisURL(t *testing.T) {
	client, err := NewClient("rediss://default:secret@example.com:6380/2", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	options := client.Options()
	assert.Equal(t, "example.com:6380", options.Addr)
	assert.Equal(t, "default", options.Username)
	assert.Equal(t, "secret", options.Password)
	assert.Equal(t, 2, options.DB)
	assert.NotNil(t, options.TLSConfig)
}

func TestNewClientRejectsInvalidManagedRedisURL(t *testing.T) {
	client, err := NewClient("://invalid", "", "")
	assert.Nil(t, client)
	assert.ErrorContains(t, err, "REDIS_URL")
}
