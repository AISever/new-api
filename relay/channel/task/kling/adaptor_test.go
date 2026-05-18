package kling

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestParseTaskResultIncludesActualDurationForBilling(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"code": 0,
		"data": {
			"task_id": "task-upstream",
			"task_status": "succeed",
			"task_result": {
				"videos": [
					{"url": "https://example.test/video.mp4", "duration": "8"}
				]
			}
		}
	}`)

	taskInfo, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	require.NotNil(t, taskInfo.BillingRequestInput)

	var billingBody map[string]interface{}
	require.NoError(t, common.Unmarshal(taskInfo.BillingRequestInput.Body, &billingBody))
	require.Equal(t, float64(8), billingBody["duration"])
}
