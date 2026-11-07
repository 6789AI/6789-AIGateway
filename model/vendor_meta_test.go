package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDeleteVendorIfUnusedPreservesAssignedModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Vendor{}, &Model{}))
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	vendor := Vendor{Name: "Muse", Status: 1}
	require.NoError(t, db.Create(&vendor).Error)
	assigned := Model{ModelName: "video-model", VendorID: vendor.Id, Status: 1}
	require.NoError(t, db.Create(&assigned).Error)

	err = DeleteVendorIfUnused(vendor.Id)
	require.ErrorIs(t, err, ErrVendorInUse)
	var stored Vendor
	require.NoError(t, db.First(&stored, vendor.Id).Error)
	var model Model
	require.NoError(t, db.First(&model, assigned.Id).Error)
	assert.Equal(t, vendor.Id, model.VendorID)

	require.NoError(t, db.Delete(&model).Error)
	require.NoError(t, DeleteVendorIfUnused(vendor.Id))
	assert.ErrorIs(t, db.First(&stored, vendor.Id).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, DeleteVendorIfUnused(vendor.Id), gorm.ErrRecordNotFound)
}
