package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSearchRedemptionsFiltersAndPaginates(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	now := common.GetTimestamp()
	redemptions := []Redemption{
		{Id: 1, Name: "alpha-active", Key: "00000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 0},
		{Id: 2, Name: "alpha-future", Key: "00000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now + 3600},
		{Id: 3, Name: "alpha-expired", Key: "00000000000000000000000000000003", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now - 10},
		{Id: 4, Name: "beta-disabled", Key: "00000000000000000000000000000004", Status: common.RedemptionCodeStatusDisabled, ExpiredTime: 0},
		{Id: 5, Name: "beta-used", Key: "00000000000000000000000000000005", Status: common.RedemptionCodeStatusUsed, ExpiredTime: 0},
	}
	require.NoError(t, DB.Create(&redemptions).Error)

	tests := []struct {
		name      string
		keyword   string
		status    string
		startIdx  int
		num       int
		wantTotal int64
		wantIds   []int
	}{
		{
			name:      "no filters returns all rows",
			num:       10,
			wantTotal: 5,
			wantIds:   []int{5, 4, 3, 2, 1},
		},
		{
			name:      "keyword filters by name prefix",
			keyword:   "alpha",
			num:       10,
			wantTotal: 3,
			wantIds:   []int{3, 2, 1},
		},
		{
			name:      "enabled status excludes expired rows",
			status:    "1",
			num:       10,
			wantTotal: 2,
			wantIds:   []int{2, 1},
		},
		{
			name:      "expired status returns enabled expired rows",
			status:    "expired",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{3},
		},
		{
			name:      "disabled status",
			status:    "2",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{4},
		},
		{
			name:      "used status",
			status:    "3",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{5},
		},
		{
			name:      "pagination keeps unpaged total",
			startIdx:  1,
			num:       2,
			wantTotal: 5,
			wantIds:   []int{4, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := SearchRedemptions(tt.keyword, tt.status, tt.startIdx, tt.num)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			gotIds := make([]int, 0, len(rows))
			for _, row := range rows {
				gotIds = append(gotIds, row.Id)
			}
			assert.Equal(t, tt.wantIds, gotIds)
		})
	}
}

func setupRedeemFixture(t *testing.T, quota int) (userId int, key string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0}
	require.NoError(t, DB.Create(user).Error)

	key = "10000000000000000000000000000001"
	redemption := &Redemption{
		Name:        "redeem-test",
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       quota,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)
	return user.Id, key
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500)

	quota, err := Redeem(key, userId)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
}

// Exactly one of several concurrent redeems of the same code may win, and
// quota must be credited exactly once.
func TestRedeemConcurrentSingleSuccess(t *testing.T) {
	userId, key := setupRedeemFixture(t, 300)

	const goroutines = 5
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, userId); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redeem should succeed")

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 300, user.Quota, "quota must be credited exactly once")
}

func TestRedeemRollsBackWhenUserDoesNotExist(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}))

	key := "10000000000000000000000000000002"
	redemption := &Redemption{
		Name:        "redeem-missing-user",
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       500,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)

	_, err := Redeem(key, 999999)
	require.ErrorIs(t, err, ErrRedeemFailed)

	var stored Redemption
	require.NoError(t, DB.First(&stored, "id = ?", redemption.Id).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, stored.Status)
}

func TestRedemptionBatchAllowsOneCodePerUser(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &RedemptionBatchClaim{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionBatchClaim{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionBatchClaim{}).Error)
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	users := []User{
		{Username: "batch-redeem-user-1", Password: "password", Status: common.UserStatusEnabled, AffCode: "batch-user-1"},
		{Username: "batch-redeem-user-2", Password: "password", Status: common.UserStatusEnabled, AffCode: "batch-user-2"},
	}
	require.NoError(t, DB.Create(&users).Error)
	batch := Redemption{Name: "gift", Count: 2, Quota: 250, OnePerUser: true, CreatedTime: common.GetTimestamp()}
	keys, err := CreateRedemptionCodes(batch)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	var codes []Redemption
	require.NoError(t, DB.Where("key IN ?", keys).Find(&codes).Error)
	require.Len(t, codes, 2)
	assert.NotEmpty(t, codes[0].BatchID)
	assert.Equal(t, codes[0].BatchID, codes[1].BatchID)
	assert.True(t, codes[0].OnePerUser)

	quota, err := Redeem(keys[0], users[0].Id)
	require.NoError(t, err)
	assert.Equal(t, 250, quota)
	deleted, err := DeleteInvalidRedemptions()
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
	_, err = Redeem(keys[1], users[0].Id)
	require.ErrorIs(t, err, ErrRedeemFailed)

	var unused Redemption
	require.NoError(t, DB.Where("key = ?", keys[1]).First(&unused).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, unused.Status)
	var firstUser User
	require.NoError(t, DB.First(&firstUser, users[0].Id).Error)
	assert.Equal(t, 250, firstUser.Quota)
	var claimCount int64
	require.NoError(t, DB.Model(&RedemptionBatchClaim{}).Where("batch_id = ? AND user_id = ?", codes[0].BatchID, users[0].Id).Count(&claimCount).Error)
	assert.EqualValues(t, 1, claimCount)

	_, err = Redeem(keys[1], users[1].Id)
	require.NoError(t, err)
	otherBatchKeys, err := CreateRedemptionCodes(Redemption{Name: "gift", Count: 1, Quota: 250, OnePerUser: true})
	require.NoError(t, err)
	_, err = Redeem(otherBatchKeys[0], users[0].Id)
	require.NoError(t, err)
	unrestrictedKeys, err := CreateRedemptionCodes(Redemption{Name: "gift", Count: 1, Quota: 250})
	require.NoError(t, err)
	_, err = Redeem(unrestrictedKeys[0], users[0].Id)
	require.NoError(t, err)
	require.NoError(t, DB.First(&firstUser, users[0].Id).Error)
	assert.Equal(t, 750, firstUser.Quota)

	rollbackKeys, err := CreateRedemptionCodes(Redemption{Name: "gift", Count: 1, Quota: 250, OnePerUser: true})
	require.NoError(t, err)
	_, err = Redeem(rollbackKeys[0], 999999)
	require.ErrorIs(t, err, ErrRedeemFailed)
	var rollbackCode Redemption
	require.NoError(t, DB.Where("key = ?", rollbackKeys[0]).First(&rollbackCode).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, rollbackCode.Status)
	require.NoError(t, DB.Model(&RedemptionBatchClaim{}).Where("batch_id = ?", rollbackCode.BatchID).Count(&claimCount).Error)
	assert.Zero(t, claimCount)
}

func TestRedemptionBatchConcurrentCodesCreditOnce(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &RedemptionBatchClaim{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionBatchClaim{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionBatchClaim{}).Error)
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})
	user := User{Username: "concurrent-batch-user", Password: "password", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)
	keys, err := CreateRedemptionCodes(Redemption{Name: "gift", Count: 2, Quota: 300, OnePerUser: true})
	require.NoError(t, err)

	results := make(chan error, 2)
	for _, key := range keys {
		go func(key string) {
			_, redeemErr := Redeem(key, user.Id)
			results <- redeemErr
		}(key)
	}
	successes := 0
	for range keys {
		if err := <-results; err == nil {
			successes++
		}
	}
	assert.Equal(t, 1, successes)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, 300, user.Quota)
	var claimCount int64
	require.NoError(t, DB.Model(&RedemptionBatchClaim{}).Count(&claimCount).Error)
	assert.EqualValues(t, 1, claimCount)
}
