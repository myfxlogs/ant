package secrets_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"anttrader/internal/secrets"
)

func TestEnvMasterKey_Empty(t *testing.T) {
	t.Setenv("ANT_MASTER_KEY", "")
	_, err := secrets.EnvMasterKey{}.MasterKey(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestEnvMasterKey_Valid(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	assert.NoError(t, err)
	t.Setenv("ANT_MASTER_KEY", key)
	k, err := secrets.EnvMasterKey{}.MasterKey(context.Background())
	assert.NoError(t, err)
	assert.Len(t, k, 32)
}

func TestEnvMasterKey_Rotate(t *testing.T) {
	_, _, err := secrets.EnvMasterKey{}.Rotate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rotate not supported")
}

func TestFileMasterKey_NotSet(t *testing.T) {
	t.Setenv("ANT_MASTER_KEY_FILE", "")
	_, err := secrets.FileMasterKey{}.MasterKey(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestFileMasterKey_Valid(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	assert.NoError(t, err)
	f, err := os.CreateTemp("", "ant-key-*")
	assert.NoError(t, err)
	defer os.Remove(f.Name())
	_, err = f.WriteString(key)
	assert.NoError(t, err)
	f.Close()

	t.Setenv("ANT_MASTER_KEY_FILE", f.Name())
	k, err := secrets.FileMasterKey{}.MasterKey(context.Background())
	assert.NoError(t, err)
	assert.Len(t, k, 32)
}

func TestEnvMasterKey_Base64Format(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	assert.NoError(t, err)
	t.Setenv("ANT_MASTER_KEY", key)
	k, err := secrets.EnvMasterKey{}.MasterKey(context.Background())
	assert.NoError(t, err)
	assert.Len(t, k, 32)
}

func TestFileMasterKey_Rotate(t *testing.T) {
	_, _, err := secrets.FileMasterKey{}.Rotate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rotate not supported")
}
