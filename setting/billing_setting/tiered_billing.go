package billing_setting

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio      = "ratio"
	BillingModeTieredExpr = "tiered_expr"
	BillingModeScheduled  = "scheduled_price"
	BillingModeField      = "billing_mode"
	BillingExprField      = "billing_expr"
	PriceSchedulesField   = "price_schedules"
	FreeModelBannerField  = "free_model_banner_enabled"

	PriceScheduleAbsolute = "absolute"
	PriceScheduleWeekly   = "weekly"
	maxSchedulesPerModel  = 64
)

type PriceSchedule struct {
	ID          string   `json:"id,omitempty"`
	Type        string   `json:"type"`
	Price       *float64 `json:"price"`
	StartAt     int64    `json:"start_at,omitempty"`
	EndAt       int64    `json:"end_at,omitempty"`
	Weekdays    []int    `json:"weekdays,omitempty"`
	StartMinute int      `json:"start_minute,omitempty"`
	EndMinute   int      `json:"end_minute,omitempty"`
	Timezone    string   `json:"timezone,omitempty"`
	ShowBanner  *bool    `json:"show_banner,omitempty"`
}

type FreeModelPromotion struct {
	ModelName string `json:"model_name"`
	EndsAt    int64  `json:"ends_at,omitempty"`
}

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr,
// billing_setting.price_schedules
type BillingSetting struct {
	BillingMode            map[string]string          `json:"billing_mode"`
	BillingExpr            map[string]string          `json:"billing_expr"`
	PriceSchedules         map[string][]PriceSchedule `json:"price_schedules"`
	FreeModelBannerEnabled bool                       `json:"free_model_banner_enabled"`
}

var billingSetting = BillingSetting{
	BillingMode:            make(map[string]string),
	BillingExpr:            make(map[string]string),
	PriceSchedules:         make(map[string][]PriceSchedule),
	FreeModelBannerEnabled: true,
}

var priceScheduleLocations sync.Map

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	return lo.Assign(billingSetting.BillingExpr)
}

func GetPriceSchedules(model string) []PriceSchedule {
	rules := billingSetting.PriceSchedules[model]
	result := append([]PriceSchedule(nil), rules...)
	for index := range result {
		result[index].Weekdays = append([]int(nil), result[index].Weekdays...)
	}
	return result
}

func GetPriceSchedulesCopy() map[string][]PriceSchedule {
	result := make(map[string][]PriceSchedule, len(billingSetting.PriceSchedules))
	for model := range billingSetting.PriceSchedules {
		result[model] = GetPriceSchedules(model)
	}
	return result
}

func IsFreeModelBannerEnabled() bool {
	return billingSetting.FreeModelBannerEnabled
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 3)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	if schedules := GetPriceSchedulesCopy(); len(schedules) > 0 {
		extra[PriceSchedulesField] = schedules
	}
	return lo.Assign(base, extra)
}

func ValidatePriceSchedulesJSON(value string) error {
	var schedules map[string][]PriceSchedule
	if err := common.UnmarshalJsonStr(value, &schedules); err != nil {
		return fmt.Errorf("invalid price schedules: %w", err)
	}
	for model, rules := range schedules {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("model name cannot be empty")
		}
		if len(rules) > maxSchedulesPerModel {
			return fmt.Errorf("model %s has more than %d price schedules", model, maxSchedulesPerModel)
		}
		ids := make(map[string]struct{}, len(rules))
		for index, rule := range rules {
			if err := validatePriceSchedule(rule); err != nil {
				return fmt.Errorf("model %s schedule %d: %w", model, index+1, err)
			}
			if rule.ID != "" {
				if _, exists := ids[rule.ID]; exists {
					return fmt.Errorf("model %s has duplicate schedule id %q", model, rule.ID)
				}
				ids[rule.ID] = struct{}{}
			}
		}
	}
	return nil
}

func validatePriceSchedule(rule PriceSchedule) error {
	if rule.Price == nil {
		return fmt.Errorf("price is required")
	}
	if *rule.Price < 0 || math.IsNaN(*rule.Price) || math.IsInf(*rule.Price, 0) {
		return fmt.Errorf("price must be a finite number greater than or equal to zero")
	}

	switch rule.Type {
	case PriceScheduleAbsolute:
		if rule.StartAt <= 0 || rule.EndAt <= rule.StartAt {
			return fmt.Errorf("absolute schedule must have a valid start and end time")
		}
	case PriceScheduleWeekly:
		if len(rule.Weekdays) == 0 {
			return fmt.Errorf("weekly schedule must select at least one weekday")
		}
		seen := make(map[int]struct{}, len(rule.Weekdays))
		for _, weekday := range rule.Weekdays {
			if weekday < 0 || weekday > 6 {
				return fmt.Errorf("weekday must be between 0 and 6")
			}
			if _, exists := seen[weekday]; exists {
				return fmt.Errorf("weekday %d is duplicated", weekday)
			}
			seen[weekday] = struct{}{}
		}
		if rule.StartMinute < 0 || rule.StartMinute > 1439 || rule.EndMinute < 0 || rule.EndMinute > 1439 {
			return fmt.Errorf("weekly time must be between 00:00 and 23:59")
		}
		if strings.TrimSpace(rule.Timezone) == "" {
			return fmt.Errorf("timezone is required")
		}
		if _, err := loadPriceScheduleLocation(rule.Timezone); err != nil {
			return fmt.Errorf("invalid timezone %q: %w", rule.Timezone, err)
		}
	default:
		return fmt.Errorf("unknown schedule type %q", rule.Type)
	}
	return nil
}

func loadPriceScheduleLocation(name string) (*time.Location, error) {
	if cached, ok := priceScheduleLocations.Load(name); ok {
		return cached.(*time.Location), nil
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	priceScheduleLocations.Store(name, location)
	return location, nil
}

func localScheduleBoundary(local time.Time, dayOffset, minute int) time.Time {
	day := local.AddDate(0, 0, dayOffset)
	return time.Date(day.Year(), day.Month(), day.Day(), minute/60, minute%60, 0, 0, local.Location())
}

func priceScheduleActiveUntil(rule PriceSchedule, now time.Time) (int64, bool) {
	switch rule.Type {
	case PriceScheduleAbsolute:
		unix := now.Unix()
		return rule.EndAt, rule.StartAt <= unix && unix < rule.EndAt
	case PriceScheduleWeekly:
		location, err := loadPriceScheduleLocation(rule.Timezone)
		if err != nil || len(rule.Weekdays) == 0 {
			return 0, false
		}

		local := now.In(location)
		minute := local.Hour()*60 + local.Minute()
		weekday := int(local.Weekday())
		selectedToday := lo.Contains(rule.Weekdays, weekday)

		if rule.StartMinute == rule.EndMinute {
			if !selectedToday {
				return 0, false
			}
			for offset := 1; offset <= 7; offset++ {
				nextWeekday := (weekday + offset) % 7
				if !lo.Contains(rule.Weekdays, nextWeekday) {
					return localScheduleBoundary(local, offset, 0).Unix(), true
				}
			}
			return 0, true
		}

		if rule.StartMinute < rule.EndMinute {
			if !selectedToday || minute < rule.StartMinute || minute >= rule.EndMinute {
				return 0, false
			}
			return localScheduleBoundary(local, 0, rule.EndMinute).Unix(), true
		}

		if selectedToday && minute >= rule.StartMinute {
			return localScheduleBoundary(local, 1, rule.EndMinute).Unix(), true
		}
		previousWeekday := (weekday + 6) % 7
		if lo.Contains(rule.Weekdays, previousWeekday) && minute < rule.EndMinute {
			return localScheduleBoundary(local, 0, rule.EndMinute).Unix(), true
		}
	}
	return 0, false
}

// GetScheduledPrice returns the lowest active promotional price. The base
// price remains stored separately and is restored automatically outside rules.
func GetScheduledPrice(model string, now time.Time) (float64, bool) {
	rules := billingSetting.PriceSchedules[model]
	var effective float64
	matched := false
	for _, rule := range rules {
		if rule.Price == nil || *rule.Price < 0 || math.IsNaN(*rule.Price) || math.IsInf(*rule.Price, 0) {
			continue
		}
		if _, active := priceScheduleActiveUntil(rule, now); active && (!matched || *rule.Price < effective) {
			effective = *rule.Price
			matched = true
		}
	}
	return effective, matched
}

func GetActiveFreeModelPromotions(now time.Time) []FreeModelPromotion {
	if !billingSetting.FreeModelBannerEnabled {
		return nil
	}

	promotions := make([]FreeModelPromotion, 0)
	for model, rules := range billingSetting.PriceSchedules {
		if GetBillingMode(model) != BillingModeScheduled {
			continue
		}

		active := false
		activeIndefinitely := false
		var endsAt int64
		for _, rule := range rules {
			if rule.Price == nil || *rule.Price != 0 || (rule.ShowBanner != nil && !*rule.ShowBanner) {
				continue
			}
			ruleEndsAt, ruleActive := priceScheduleActiveUntil(rule, now)
			if !ruleActive {
				continue
			}
			active = true
			if ruleEndsAt == 0 {
				activeIndefinitely = true
				continue
			}
			if ruleEndsAt > endsAt {
				endsAt = ruleEndsAt
			}
		}
		if active {
			if activeIndefinitely {
				endsAt = 0
			}
			promotions = append(promotions, FreeModelPromotion{ModelName: model, EndsAt: endsAt})
		}
	}

	sort.Slice(promotions, func(i, j int) bool {
		return promotions[i].ModelName < promotions[j].ModelName
	})
	return promotions
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}
