package firebase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevelopmentTokensHaveExplicitRoles(t *testing.T) {
	client, err := NewAuthClient(context.Background(), true, "")
	require.NoError(t, err)

	customer, err := client.VerifyIDToken(context.Background(), "customer-1")
	require.NoError(t, err)
	assert.Equal(t, "customer-1", customer.UID)
	assert.Equal(t, "customer", customer.Role)

	admin, err := client.VerifyIDToken(context.Background(), "admin:dashboard")
	require.NoError(t, err)
	assert.Equal(t, "dashboard", admin.UID)
	assert.Equal(t, "admin", admin.Role)
}

func TestDevelopmentTokenRejectsEmptyUserID(t *testing.T) {
	client, err := NewAuthClient(context.Background(), true, "")
	require.NoError(t, err)

	_, err = client.VerifyIDToken(context.Background(), "admin:")
	assert.Error(t, err)
}
