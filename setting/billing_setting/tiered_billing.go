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
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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
	PriceAdjustmentFixed  = "fixed_price"
	PriceAdjustmentRate   = "discount"
	PromotionTypeFree     = "free"
	PromotionTypeDiscount = "discount"
	PromotionTypeFixed    = "fixed_price"
	maxSchedulesPerModel  = 64
)

type PriceSchedule struct {
	ID             string   `json:"id,omitempty"`
	Type           string   `json:"type"`
	AdjustmentType string   `json:"adjustment_type,omitempty"`
	Price          *float64 `json:"price,omitempty"`
	DiscountRate   *float64 `json:"discount_rate,omitempty"`
	StartAt        int64    `json:"start_at,omitempty"`
	EndAt          int64    `json:"end_at,omitempty"`
	Weekdays       []int    `json:"weekdays,omitempty"`
	StartMinute    int      `json:"start_minute,omitempty"`
	EndMinute      int      `json:"end_minute,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	ShowBanner     *bool    `json:"show_banner,omitempty"`
}

type ModelPromotion struct {
	ModelName         string   `json:"model_name"`
	PromotionType     string   `json:"promotion_type"`
	Price             *float64 `json:"price,omitempty"`
	DiscountRate      *float64 `json:"discount_rate,omitempty"`
	EndsAt            int64    `json:"ends_at,omitempty"`
	ActivityKey       string   `json:"-"`
	legacyActivityKey string   `json:"-"`
}

// ScheduledAdjustment is the selected active promotion and the shared
// occurrence key used to enforce a per-user activity allowance.
type ScheduledAdjustment struct {
	Value       float64
	ActivityKey string
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
	switch priceScheduleAdjustmentType(rule) {
	case PriceAdjustmentFixed:
		if rule.Price == nil {
			return fmt.Errorf("price is required")
		}
		if *rule.Price < 0 || math.IsNaN(*rule.Price) || math.IsInf(*rule.Price, 0) {
			return fmt.Errorf("price must be a finite number greater than or equal to zero")
		}
	case PriceAdjustmentRate:
		if rule.DiscountRate == nil {
			return fmt.Errorf("discount rate is required")
		}
		if *rule.DiscountRate < 0 || *rule.DiscountRate > 1 || math.IsNaN(*rule.DiscountRate) || math.IsInf(*rule.DiscountRate, 0) {
			return fmt.Errorf("discount rate must be a finite number between zero and one")
		}
	default:
		return fmt.Errorf("unknown adjustment type %q", rule.AdjustmentType)
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

func priceScheduleAdjustmentType(rule PriceSchedule) string {
	if rule.AdjustmentType == "" {
		return PriceAdjustmentFixed
	}
	return rule.AdjustmentType
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

func priceScheduleWeeklyKeyParts(rule PriceSchedule) string {
	weekdays := append([]int(nil), rule.Weekdays...)
	sort.Ints(weekdays)
	weekdayParts := make([]string, len(weekdays))
	for index, weekday := range weekdays {
		weekdayParts[index] = fmt.Sprintf("%d", weekday)
	}
	return strings.Join(weekdayParts, ",")
}

func priceScheduleLegacyActivityKey(rule PriceSchedule, endsAt int64) string {
	if rule.Type != PriceScheduleWeekly {
		return ""
	}
	return fmt.Sprintf(
		"v1:weekly:%s:%s:%d:%d:%d",
		rule.Timezone,
		priceScheduleWeeklyKeyParts(rule),
		rule.StartMinute,
		rule.EndMinute,
		endsAt,
	)
}

func priceScheduleActivityKey(rule PriceSchedule, endsAt int64, now time.Time) string {
	switch rule.Type {
	case PriceScheduleAbsolute:
		return fmt.Sprintf("v1:absolute:%d:%d", rule.StartAt, rule.EndAt)
	case PriceScheduleWeekly:
		weekdayParts := priceScheduleWeeklyKeyParts(rule)
		if rule.StartMinute == rule.EndMinute {
			// An all-day rule can span consecutive selected weekdays. Include the
			// local calendar date so each day's activity gets its own allowance.
			if location, err := loadPriceScheduleLocation(rule.Timezone); err == nil {
				local := now.In(location)
				occurrenceDate := local.Format("2006-01-02")
				return fmt.Sprintf(
					"v2:weekly:%s:%s:%d:%d:%d:%s",
					rule.Timezone,
					weekdayParts,
					rule.StartMinute,
					rule.EndMinute,
					endsAt,
					occurrenceDate,
				)
			}
		}
		return priceScheduleLegacyActivityKey(rule, endsAt)
	default:
		return ""
	}
}

// GetScheduledPriceAdjustment returns the lowest active scheduled price and
// a key shared by models configured for the same activity occurrence.
func GetScheduledPriceAdjustment(model string, basePrice float64, now time.Time) (ScheduledAdjustment, bool) {
	rules := billingSetting.PriceSchedules[model]
	var selected ScheduledAdjustment
	matched := false
	for _, rule := range rules {
		endsAt, active := priceScheduleActiveUntil(rule, now)
		if !active {
			continue
		}

		var candidate float64
		switch priceScheduleAdjustmentType(rule) {
		case PriceAdjustmentFixed:
			if rule.Price == nil || *rule.Price < 0 || math.IsNaN(*rule.Price) || math.IsInf(*rule.Price, 0) {
				continue
			}
			candidate = *rule.Price
		case PriceAdjustmentRate:
			if rule.DiscountRate == nil || *rule.DiscountRate < 0 || *rule.DiscountRate > 1 || math.IsNaN(*rule.DiscountRate) || math.IsInf(*rule.DiscountRate, 0) {
				continue
			}
			candidate = basePrice * *rule.DiscountRate
		default:
			continue
		}

		if !matched || candidate < selected.Value {
			selected = ScheduledAdjustment{
				Value:       candidate,
				ActivityKey: priceScheduleActivityKey(rule, endsAt, now),
			}
			matched = true
		}
	}
	return selected, matched
}

// GetScheduledPrice returns the lowest active price after applying fixed-price
// and percentage-discount rules to the supplied base price.
func GetScheduledPrice(model string, basePrice float64, now time.Time) (float64, bool) {
	adjustment, matched := GetScheduledPriceAdjustment(model, basePrice, now)
	return adjustment.Value, matched
}

// GetScheduledDiscountAdjustment returns the lowest active discount for token
// and expression billing. Fixed-price rules are intentionally ignored.
func GetScheduledDiscountAdjustment(model string, now time.Time) (ScheduledAdjustment, bool) {
	rules := billingSetting.PriceSchedules[model]
	var selected ScheduledAdjustment
	matched := false
	for _, rule := range rules {
		if priceScheduleAdjustmentType(rule) != PriceAdjustmentRate || rule.DiscountRate == nil {
			continue
		}
		rate := *rule.DiscountRate
		if rate < 0 || rate > 1 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			continue
		}
		endsAt, active := priceScheduleActiveUntil(rule, now)
		if active && (!matched || rate < selected.Value) {
			selected = ScheduledAdjustment{
				Value:       rate,
				ActivityKey: priceScheduleActivityKey(rule, endsAt, now),
			}
			matched = true
		}
	}
	return selected, matched
}

// GetScheduledDiscount returns the lowest active discount rate for token and
// expression billing. Fixed-price rules are intentionally ignored.
func GetScheduledDiscount(model string, now time.Time) (float64, bool) {
	adjustment, matched := GetScheduledDiscountAdjustment(model, now)
	return adjustment.Value, matched
}

func promotionFromRule(model string, rule PriceSchedule, endsAt int64, now time.Time) (ModelPromotion, bool) {
	promotion := ModelPromotion{
		ModelName:   model,
		EndsAt:      endsAt,
		ActivityKey: priceScheduleActivityKey(rule, endsAt, now),
	}
	if rule.Type == PriceScheduleWeekly && rule.StartMinute == rule.EndMinute {
		promotion.legacyActivityKey = priceScheduleLegacyActivityKey(rule, endsAt)
	}
	switch priceScheduleAdjustmentType(rule) {
	case PriceAdjustmentFixed:
		if rule.Price == nil || *rule.Price < 0 || math.IsNaN(*rule.Price) || math.IsInf(*rule.Price, 0) {
			return ModelPromotion{}, false
		}
		price := *rule.Price
		promotion.Price = &price
		if price == 0 {
			promotion.PromotionType = PromotionTypeFree
		} else {
			promotion.PromotionType = PromotionTypeFixed
		}
	case PriceAdjustmentRate:
		if rule.DiscountRate == nil || *rule.DiscountRate < 0 || *rule.DiscountRate > 1 || math.IsNaN(*rule.DiscountRate) || math.IsInf(*rule.DiscountRate, 0) {
			return ModelPromotion{}, false
		}
		rate := *rule.DiscountRate
		promotion.DiscountRate = &rate
		if rate == 0 {
			promotion.PromotionType = PromotionTypeFree
		} else {
			promotion.PromotionType = PromotionTypeDiscount
		}
	default:
		return ModelPromotion{}, false
	}
	return promotion, true
}

func promotionRank(promotion ModelPromotion, basePrice float64, hasBasePrice bool) (int, float64) {
	switch promotion.PromotionType {
	case PromotionTypeFree:
		return 0, 0
	case PromotionTypeDiscount:
		if promotion.DiscountRate != nil {
			if hasBasePrice {
				return 1, basePrice * *promotion.DiscountRate
			}
			return 1, *promotion.DiscountRate
		}
	case PromotionTypeFixed:
		if promotion.Price != nil {
			if hasBasePrice {
				return 1, *promotion.Price
			}
			return 2, *promotion.Price
		}
	}
	return 3, math.MaxFloat64
}

func promotionBetter(candidate, current ModelPromotion, basePrice float64, hasBasePrice bool) bool {
	candidateRank, candidateValue := promotionRank(candidate, basePrice, hasBasePrice)
	currentRank, currentValue := promotionRank(current, basePrice, hasBasePrice)
	return candidateRank < currentRank || (candidateRank == currentRank && candidateValue < currentValue)
}

func samePromotion(candidate, current ModelPromotion) bool {
	if candidate.PromotionType != current.PromotionType {
		return candidate.PromotionType == PromotionTypeFree && current.PromotionType == PromotionTypeFree
	}
	switch candidate.PromotionType {
	case PromotionTypeFree:
		return true
	case PromotionTypeDiscount:
		return candidate.DiscountRate != nil && current.DiscountRate != nil &&
			*candidate.DiscountRate == *current.DiscountRate
	case PromotionTypeFixed:
		return candidate.Price != nil && current.Price != nil && *candidate.Price == *current.Price
	default:
		return false
	}
}

func getActiveModelPromotions(now time.Time, bannerOnly bool) []ModelPromotion {
	promotions := make([]ModelPromotion, 0)
	for model, rules := range billingSetting.PriceSchedules {
		var selected ModelPromotion
		matched := false
		selectedVisible := false
		basePrice, hasBasePrice := ratio_setting.GetModelPrice(model, false)
		for _, rule := range rules {
			if priceScheduleAdjustmentType(rule) == PriceAdjustmentFixed && GetBillingMode(model) != BillingModeScheduled {
				continue
			}
			ruleEndsAt, ruleActive := priceScheduleActiveUntil(rule, now)
			if !ruleActive {
				continue
			}
			candidate, ok := promotionFromRule(model, rule, ruleEndsAt, now)
			if !ok {
				continue
			}
			visible := rule.ShowBanner == nil || *rule.ShowBanner
			if !matched || promotionBetter(candidate, selected, basePrice, hasBasePrice) {
				selected = candidate
				matched = true
				selectedVisible = visible
				continue
			}
			if !samePromotion(candidate, selected) || !visible {
				continue
			}
			if !selectedVisible {
				selected.EndsAt = candidate.EndsAt
				selectedVisible = true
				continue
			}
			if selected.EndsAt != 0 && (candidate.EndsAt == 0 || candidate.EndsAt > selected.EndsAt) {
				selected.EndsAt = candidate.EndsAt
			}
		}
		if matched && (!bannerOnly || selectedVisible) {
			promotions = append(promotions, selected)
		}
	}

	sort.Slice(promotions, func(i, j int) bool {
		return promotions[i].ModelName < promotions[j].ModelName
	})
	return promotions
}

// GetActiveModelPromotions returns active pricing activities independently of
// whether the optional global promotion banner is enabled.
func GetActiveModelPromotions(now time.Time) []ModelPromotion {
	return getActiveModelPromotions(now, false)
}

// GetActivePromotionActivityKeys returns each active pricing activity once,
// even when multiple models share the same occurrence and allowance.
func GetActivePromotionActivityKeys(now time.Time) []string {
	activityKeys := make(map[string]struct{})
	for _, promotion := range GetActiveModelPromotions(now) {
		if promotion.ActivityKey != "" {
			activityKeys[promotion.ActivityKey] = struct{}{}
		}
		if promotion.legacyActivityKey != "" {
			activityKeys[promotion.legacyActivityKey] = struct{}{}
		}
	}
	keys := make([]string, 0, len(activityKeys))
	for key := range activityKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func GetActiveBannerModelPromotions(now time.Time) []ModelPromotion {
	if !billingSetting.FreeModelBannerEnabled {
		return make([]ModelPromotion, 0)
	}
	return getActiveModelPromotions(now, true)
}

// GetActiveFreeModelPromotions is kept for callers that only need the legacy
// free banner subset.
func GetActiveFreeModelPromotions(now time.Time) []ModelPromotion {
	return lo.Filter(GetActiveBannerModelPromotions(now), func(promotion ModelPromotion, _ int) bool {
		return promotion.PromotionType == PromotionTypeFree
	})
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
