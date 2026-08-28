package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
)

func TestPromotionFreeUsageIsIncludedInConsumptionLogMetadata(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{PromotionFreeUsage: true}

	other := GenerateMjOtherInfo(relayInfo, types.PriceData{})

	assert.Equal(t, true, other["promotion_free_usage"])
}

func TestPromotionFreeUsageIsOmittedFromRegularConsumptionLogMetadata(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{}
	other := make(map[string]interface{})

	appendBillingInfo(relayInfo, other)

	assert.NotContains(t, other, "promotion_free_usage")
}
