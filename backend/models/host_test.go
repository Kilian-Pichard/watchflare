package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHost_BeforeCreate_GeneratesUUID(t *testing.T) {
	h := &Host{}
	err := h.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, h.ID)
}

func TestHost_BeforeCreate_PreservesExistingID(t *testing.T) {
	h := &Host{ID: "existing-id"}
	err := h.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.Equal(t, "existing-id", h.ID)
}

func TestHost_TableName(t *testing.T) {
	assert.Equal(t, "hosts", Host{}.TableName())
}
