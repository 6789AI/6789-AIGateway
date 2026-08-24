package billing_setting

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pricePointer(value float64) *float64 {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func TestGetScheduledPrice(t *testing.T) {
	original := billingSetting.PriceSchedules
	t.Cleanup(func() {
		billingSetting.PriceSchedules = original
	})

	tests := []struct {
		name      string
		rules     []PriceSchedule
		now       time.Time
		wantPrice float64
		wantMatch bool
	}{
		{
			name: "absolute range includes start and excludes end",
			rules: []PriceSchedule{{
				Type: PriceScheduleAbsolute, Price: pricePointer(0), StartAt: 100, EndAt: 200,
			}},
			now:       time.Unix(100, 0),
			wantPrice: 0,
			wantMatch: true,
		},
		{
			name: "absolute range has ended",
			rules: []PriceSchedule{{
				Type: PriceScheduleAbsolute, Price: pricePointer(0.2), StartAt: 100, EndAt: 200,
			}},
			now:       time.Unix(200, 0),
			wantMatch: false,
		},
		{
			name: "weekly daytime range",
			rules: []PriceSchedule{{
				Type: PriceScheduleWeekly, Price: pricePointer(0.3), Weekdays: []int{1},
				StartMinute: 9 * 60, EndMinute: 17 * 60, Timezone: "UTC",
			}},
			now:       time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC),
			wantPrice: 0.3,
			wantMatch: true,
		},
		{
			name: "weekly overnight range continues on next day",
			rules: []PriceSchedule{{
				Type: PriceScheduleWeekly, Price: pricePointer(0.4), Weekdays: []int{1},
				StartMinute: 22 * 60, EndMinute: 2 * 60, Timezone: "UTC",
			}},
			now:       time.Date(2026, time.August, 25, 1, 59, 0, 0, time.UTC),
			wantPrice: 0.4,
			wantMatch: true,
		},
		{
			name: "equal weekly times mean all day",
			rules: []PriceSchedule{{
				Type: PriceScheduleWeekly, Price: pricePointer(0.5), Weekdays: []int{1},
				StartMinute: 0, EndMinute: 0, Timezone: "UTC",
			}},
			now:       time.Date(2026, time.August, 24, 23, 59, 0, 0, time.UTC),
			wantPrice: 0.5,
			wantMatch: true,
		},
		{
			name: "overlapping promotions use lowest price",
			rules: []PriceSchedule{
				{Type: PriceScheduleAbsolute, Price: pricePointer(0.8), StartAt: 100, EndAt: 200},
				{Type: PriceScheduleAbsolute, Price: pricePointer(0.1), StartAt: 100, EndAt: 200},
			},
			now:       time.Unix(150, 0),
			wantPrice: 0.1,
			wantMatch: true,
		},
		{
			name: "discount applies to the base price",
			rules: []PriceSchedule{{
				Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate,
				DiscountRate: pricePointer(0.8), StartAt: 100, EndAt: 200,
			}},
			now:       time.Unix(150, 0),
			wantPrice: 0.8,
			wantMatch: true,
		},
		{
			name: "overlapping fixed and discount activities use lowest effective price",
			rules: []PriceSchedule{
				{Type: PriceScheduleAbsolute, Price: pricePointer(0.7), StartAt: 100, EndAt: 200},
				{Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate, DiscountRate: pricePointer(0.5), StartAt: 100, EndAt: 200},
			},
			now:       time.Unix(150, 0),
			wantPrice: 0.5,
			wantMatch: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": test.rules}

			price, matched := GetScheduledPrice("model", 1, test.now)

			assert.Equal(t, test.wantMatch, matched)
			assert.Equal(t, test.wantPrice, price)
		})
	}
}

func TestValidatePriceSchedulesJSON(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{
			name:  "accepts free absolute and weekly prices",
			value: `{"model":[{"type":"absolute","price":0,"start_at":100,"end_at":200},{"type":"weekly","price":0,"weekdays":[1,5],"start_minute":1320,"end_minute":120,"timezone":"UTC"}]}`,
		},
		{
			name:  "accepts discount activities",
			value: `{"model":[{"type":"absolute","adjustment_type":"discount","discount_rate":0.8,"start_at":100,"end_at":200}]}`,
		},
		{
			name:    "rejects missing price",
			value:   `{"model":[{"type":"absolute","start_at":100,"end_at":200}]}`,
			wantErr: "price is required",
		},
		{
			name:    "rejects reversed absolute range",
			value:   `{"model":[{"type":"absolute","price":1,"start_at":200,"end_at":100}]}`,
			wantErr: "valid start and end time",
		},
		{
			name:    "rejects weekly rule without weekdays",
			value:   `{"model":[{"type":"weekly","price":1,"start_minute":0,"end_minute":1,"timezone":"UTC"}]}`,
			wantErr: "at least one weekday",
		},
		{
			name:    "rejects invalid timezone",
			value:   `{"model":[{"type":"weekly","price":1,"weekdays":[1],"start_minute":0,"end_minute":1,"timezone":"Mars/Olympus"}]}`,
			wantErr: "invalid timezone",
		},
		{
			name:    "rejects discount above one",
			value:   `{"model":[{"type":"absolute","adjustment_type":"discount","discount_rate":1.1,"start_at":100,"end_at":200}]}`,
			wantErr: "between zero and one",
		},
		{
			name:    "rejects unknown adjustment type",
			value:   `{"model":[{"type":"absolute","adjustment_type":"surcharge","price":1,"start_at":100,"end_at":200}]}`,
			wantErr: "unknown adjustment type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePriceSchedulesJSON(test.value)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestGetScheduledDiscount(t *testing.T) {
	original := billingSetting.PriceSchedules
	t.Cleanup(func() {
		billingSetting.PriceSchedules = original
	})

	billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": {
		{Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate, DiscountRate: pricePointer(0.8), StartAt: 100, EndAt: 200},
		{Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate, DiscountRate: pricePointer(0.5), StartAt: 100, EndAt: 200},
		{Type: PriceScheduleAbsolute, Price: pricePointer(0), StartAt: 100, EndAt: 200},
	}}

	rate, matched := GetScheduledDiscount("model", time.Unix(150, 0))

	require.True(t, matched)
	assert.Equal(t, 0.5, rate)
}

func TestGetActiveFreeModelPromotions(t *testing.T) {
	originalModes := billingSetting.BillingMode
	originalSchedules := billingSetting.PriceSchedules
	originalEnabled := billingSetting.FreeModelBannerEnabled
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.PriceSchedules = originalSchedules
		billingSetting.FreeModelBannerEnabled = originalEnabled
	})

	now := time.Date(2026, time.August, 24, 10, 30, 0, 0, time.UTC)
	billingSetting.BillingMode = map[string]string{
		"absolute-free": BillingModeScheduled,
		"hidden-free":   BillingModeScheduled,
		"paid-promo":    BillingModeScheduled,
		"stale-rule":    BillingModeRatio,
		"weekly-free":   BillingModeScheduled,
	}
	billingSetting.PriceSchedules = map[string][]PriceSchedule{
		"absolute-free": {{
			Type: PriceScheduleAbsolute, Price: pricePointer(0),
			StartAt: now.Add(-time.Hour).Unix(), EndAt: now.Add(time.Hour).Unix(),
		}},
		"hidden-free": {{
			Type: PriceScheduleAbsolute, Price: pricePointer(0), ShowBanner: boolPointer(false),
			StartAt: now.Add(-time.Hour).Unix(), EndAt: now.Add(time.Hour).Unix(),
		}},
		"paid-promo": {{
			Type: PriceScheduleAbsolute, Price: pricePointer(0.1),
			StartAt: now.Add(-time.Hour).Unix(), EndAt: now.Add(time.Hour).Unix(),
		}},
		"stale-rule": {{
			Type: PriceScheduleAbsolute, Price: pricePointer(0),
			StartAt: now.Add(-time.Hour).Unix(), EndAt: now.Add(time.Hour).Unix(),
		}},
		"weekly-free": {{
			Type: PriceScheduleWeekly, Price: pricePointer(0), Weekdays: []int{1},
			StartMinute: 9 * 60, EndMinute: 17 * 60, Timezone: "UTC",
		}},
	}
	billingSetting.FreeModelBannerEnabled = true

	promotions := GetActiveFreeModelPromotions(now)

	require.Len(t, promotions, 2)
	assert.Equal(t, "absolute-free", promotions[0].ModelName)
	assert.Equal(t, now.Add(time.Hour).Unix(), promotions[0].EndsAt)
	assert.Equal(t, "weekly-free", promotions[1].ModelName)
	assert.Equal(t, time.Date(2026, time.August, 24, 17, 0, 0, 0, time.UTC).Unix(), promotions[1].EndsAt)

	billingSetting.FreeModelBannerEnabled = false
	assert.Empty(t, GetActiveFreeModelPromotions(now))
}

func TestWeeklyAllDayPromotionEndsAfterConsecutiveSelectedDays(t *testing.T) {
	originalModes := billingSetting.BillingMode
	originalSchedules := billingSetting.PriceSchedules
	originalEnabled := billingSetting.FreeModelBannerEnabled
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.PriceSchedules = originalSchedules
		billingSetting.FreeModelBannerEnabled = originalEnabled
	})

	billingSetting.BillingMode = map[string]string{"model": BillingModeScheduled}
	billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": {{
		Type: PriceScheduleWeekly, Price: pricePointer(0), Weekdays: []int{1, 2},
		StartMinute: 0, EndMinute: 0, Timezone: "UTC",
	}}}
	billingSetting.FreeModelBannerEnabled = true

	promotions := GetActiveFreeModelPromotions(time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC))

	require.Len(t, promotions, 1)
	assert.Equal(t, time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC).Unix(), promotions[0].EndsAt)
}

func TestGetActiveModelPromotionsIncludesDiscounts(t *testing.T) {
	originalModes := billingSetting.BillingMode
	originalSchedules := billingSetting.PriceSchedules
	originalEnabled := billingSetting.FreeModelBannerEnabled
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.PriceSchedules = originalSchedules
		billingSetting.FreeModelBannerEnabled = originalEnabled
	})

	now := time.Unix(150, 0)
	billingSetting.BillingMode = map[string]string{
		"discount-model": BillingModeRatio,
		"fixed-model":    BillingModeScheduled,
	}
	billingSetting.PriceSchedules = map[string][]PriceSchedule{
		"discount-model": {{
			Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate,
			DiscountRate: pricePointer(0.8), StartAt: 100, EndAt: 200,
		}},
		"fixed-model": {{
			Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentFixed,
			Price: pricePointer(0.25), StartAt: 100, EndAt: 200,
		}},
	}
	billingSetting.FreeModelBannerEnabled = true

	promotions := GetActiveModelPromotions(now)

	require.Len(t, promotions, 2)
	assert.Equal(t, PromotionTypeDiscount, promotions[0].PromotionType)
	assert.Equal(t, 0.8, *promotions[0].DiscountRate)
	assert.Equal(t, PromotionTypeFixed, promotions[1].PromotionType)
	assert.Equal(t, 0.25, *promotions[1].Price)
}

func TestGetActiveModelPromotionsUsesLowestEffectivePerRequestPrice(t *testing.T) {
	originalModes := billingSetting.BillingMode
	originalSchedules := billingSetting.PriceSchedules
	originalEnabled := billingSetting.FreeModelBannerEnabled
	originalPrices := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.PriceSchedules = originalSchedules
		billingSetting.FreeModelBannerEnabled = originalEnabled
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(originalPrices))
	})

	now := time.Unix(150, 0)
	billingSetting.BillingMode = map[string]string{"model": BillingModeScheduled}
	billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": {
		{Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate, DiscountRate: pricePointer(0.5), StartAt: 100, EndAt: 200},
		{Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentFixed, Price: pricePointer(0.1), StartAt: 100, EndAt: 200},
	}}
	billingSetting.FreeModelBannerEnabled = true
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"model":1}`))

	promotions := GetActiveModelPromotions(now)

	require.Len(t, promotions, 1)
	assert.Equal(t, PromotionTypeFixed, promotions[0].PromotionType)
	require.NotNil(t, promotions[0].Price)
	assert.InDelta(t, 0.1, *promotions[0].Price, 1e-12)
}

func TestGetActiveModelPromotionsPreservesIndefiniteDuplicate(t *testing.T) {
	originalModes := billingSetting.BillingMode
	originalSchedules := billingSetting.PriceSchedules
	originalEnabled := billingSetting.FreeModelBannerEnabled
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.PriceSchedules = originalSchedules
		billingSetting.FreeModelBannerEnabled = originalEnabled
	})

	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	billingSetting.BillingMode = map[string]string{"model": BillingModeRatio}
	billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": {
		{
			Type: PriceScheduleWeekly, AdjustmentType: PriceAdjustmentRate,
			DiscountRate: pricePointer(0.5), Weekdays: []int{0, 1, 2, 3, 4, 5, 6},
			StartMinute: 0, EndMinute: 0, Timezone: "UTC",
		},
		{
			Type: PriceScheduleAbsolute, AdjustmentType: PriceAdjustmentRate,
			DiscountRate: pricePointer(0.5), StartAt: now.Add(-time.Hour).Unix(), EndAt: now.Add(time.Hour).Unix(),
		},
	}}
	billingSetting.FreeModelBannerEnabled = true

	promotions := GetActiveModelPromotions(now)

	require.Len(t, promotions, 1)
	assert.Equal(t, PromotionTypeDiscount, promotions[0].PromotionType)
	assert.Zero(t, promotions[0].EndsAt)
}

func TestGetPricingSyncDataIncludesPriceSchedules(t *testing.T) {
	original := billingSetting.PriceSchedules
	t.Cleanup(func() {
		billingSetting.PriceSchedules = original
	})

	billingSetting.PriceSchedules = map[string][]PriceSchedule{"model": {{
		Type: PriceScheduleAbsolute, Price: pricePointer(0), StartAt: 100, EndAt: 200,
	}}}

	data := GetPricingSyncData(map[string]any{})
	schedules, ok := data[PriceSchedulesField].(map[string][]PriceSchedule)
	require.True(t, ok)
	require.Len(t, schedules["model"], 1)
	assert.Equal(t, int64(200), schedules["model"][0].EndAt)
}
