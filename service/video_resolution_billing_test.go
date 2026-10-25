package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVideoResolutionPriceSnapshotPreservesPolicyDuringPolling(t *testing.T) {
	for _, multiply := range []bool{false, true} {
		t.Run(map[bool]string{false: "per request", true: "per second"}[multiply], func(t *testing.T) {
			snapshot := model.TaskBillingContext{Version: model.TaskBillingContextVersion, ModelPrice: 0.2, GroupRatio: 1, PerCallBilling: true, MultiplyByDuration: &multiply, ResolutionPrice: true, VideoResolution: "768p", OtherRatios: map[string]float64{"seconds": 5, "resolution": 2, "size": 1.6, "resolution-768P": 3, "video_input": 0.8}}
			stored, err := common.Marshal(snapshot)
			require.NoError(t, err)
			task := &model.Task{}
			require.NoError(t, common.Unmarshal(stored, &snapshot))
			task.PrivateData.BillingContext = &snapshot
			MergeTaskBillingRatios(task, map[string]float64{"seconds": 10, "resolution": 5, "size": 3})
			data := taskBillingContextPriceData(&snapshot)
			require.NotNil(t, data)
			expected := map[string]float64{"video_input": 0.8}
			if multiply {
				expected["seconds"] = 10
			}
			assert.Equal(t, expected, data.OtherRatios())
			assert.Equal(t, 0.2, snapshot.ModelPrice)
			other := taskBillingOther(task)
			assert.Equal(t, "768p", other["video_resolution"])
			assert.Equal(t, true, other["resolution_price"])
		})
	}
}
