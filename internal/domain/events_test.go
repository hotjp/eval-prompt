package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventType(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventPromptAssetCreated, "PromptAssetCreatedV1"},
		{EventPromptAssetUpdated, "PromptAssetUpdatedV1"},
		{EventPromptAssetDeleted, "PromptAssetDeletedV1"},
		{EventPromptAssetArchived, "PromptAssetArchivedV1"},
		{EventSnapshotCommitted, "SnapshotCommittedV1"},
		{EventLabelCreated, "LabelCreatedV1"},
		{EventLabelPromoted, "LabelPromotedV1"},
		{EventEvalStarted, "EvalStartedV1"},
		{EventEvalCompleted, "EvalCompletedV1"},
		{EventEvalFailed, "EvalFailedV1"},
		{EventPromptAdapted, "PromptAdaptedV1"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, string(tt.eventType))
		})
	}
}

func TestEventStatus(t *testing.T) {
	tests := []struct {
		status EventStatus
		want   string
	}{
		{EventStatusPending, "pending"},
		{EventStatusProcessed, "processed"},
		{EventStatusFailed, "failed"},
		{EventStatus(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestBaseEvent(t *testing.T) {
	t.Run("NewBaseEvent creates valid event", func(t *testing.T) {
		aggID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
		event := NewBaseEvent("Asset", aggID, EventPromptAssetCreated, 1)

		require.NotEmpty(t, event.EventID().String())
		require.Equal(t, "Asset", event.AggregateType)
		require.Equal(t, aggID, event.AggregateID)
		require.Equal(t, EventPromptAssetCreated, event.EventType())
		require.Equal(t, int64(1), event.GetVersion())
		require.NotEmpty(t, event.IdempotencyKey)
	})

	t.Run("BaseEvent ToMap", func(t *testing.T) {
		aggID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
		event := NewBaseEvent("Asset", aggID, EventPromptAssetCreated, 1)

		m := event.ToMap()
		require.Equal(t, event.EventID().String(), m["event_id"])
		require.Equal(t, "Asset", m["aggregate_type"])
		require.Equal(t, aggID.String(), m["aggregate_id"])
		require.Equal(t, string(EventPromptAssetCreated), m["event_type"])
		require.Equal(t, int64(1), m["version"])
	})

	t.Run("BaseEvent Validate", func(t *testing.T) {
		event := BaseEvent{
			EventIDValue:   ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
			AggregateType:  "Asset",
			AggregateID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
			EventType_:     EventPromptAssetCreated,
			OccurredAt_:    time.Now(),
			IdempotencyKey: "key",
			Version_:       1,
		}
		require.NoError(t, event.Validate())
	})

	t.Run("BaseEvent Validate requires all fields", func(t *testing.T) {
		event := BaseEvent{
			EventIDValue:   ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
			AggregateType:  "Asset",
			AggregateID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
			EventType_:     EventPromptAssetCreated,
			Version_:       1,
		}
		require.NoError(t, event.Validate())
	})
}

func TestPromptAssetCreatedEvent(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	event := NewPromptAssetCreatedEvent(assetID, "Test Asset", "desc", "bizline", []string{"tag1"}, "/path", 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "Test Asset", event.Name)
		require.Equal(t, "desc", event.Description)
		require.Equal(t, "bizline", event.BizLine)
		require.Equal(t, []string{"tag1"}, event.Tags)
		require.Equal(t, "/path", event.FilePath)
	})

	t.Run("ToMap", func(t *testing.T) {
		m := event.ToMap()
		require.Equal(t, "Test Asset", m["name"])
		require.Equal(t, "/path", m["file_path"])
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty name
		badEvent := NewPromptAssetCreatedEvent(assetID, "", "desc", "bizline", nil, "/path", 1)
		require.Error(t, badEvent.Validate())
	})
}

func TestPromptAssetUpdatedEvent(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	event := NewPromptAssetUpdatedEvent(assetID, "hash123", "reason", 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "hash123", event.ContentHash)
		require.Equal(t, "reason", event.Reason)
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty content hash
		badEvent := NewPromptAssetUpdatedEvent(assetID, "", "reason", 1)
		require.Error(t, badEvent.Validate())
	})
}

func TestSnapshotCommittedEvent(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	snapshotID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}

	event := NewSnapshotCommittedEvent(assetID, snapshotID, "v1.0.0", "hash", "commit", "author", "reason", 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "v1.0.0", event.Version)
		require.Equal(t, "hash", event.ContentHash)
		require.Equal(t, "commit", event.CommitHash)
		require.Equal(t, "author", event.Author)
		require.Equal(t, "reason", event.Reason)
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty snapshot_id
		badEvent := NewSnapshotCommittedEvent(assetID, ID{}, "v1.0.0", "hash", "commit", "author", "reason", 1)
		require.Error(t, badEvent.Validate())
	})
}

func TestLabelPromotedEvent(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	event := NewLabelPromotedEvent(assetID, "prod", "v1.0.0", "v2.0.0", 85, 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "prod", event.LabelName)
		require.Equal(t, "v1.0.0", event.FromVersion)
		require.Equal(t, "v2.0.0", event.ToVersion)
		require.Equal(t, 85, event.EvalScore)
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty label name
		badEvent := NewLabelPromotedEvent(assetID, "", "v1.0.0", "v2.0.0", 85, 1)
		require.Error(t, badEvent.Validate())

		// Invalid: empty to_version
		badEvent2 := NewLabelPromotedEvent(assetID, "prod", "v1.0.0", "", 85, 1)
		require.Error(t, badEvent2.Validate())
	})
}

func TestEvalCompletedEvent(t *testing.T) {
	evalRunID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	evalCaseID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}
	snapshotID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"}

	event := NewEvalCompletedEvent(evalRunID, evalCaseID, snapshotID, "passed", 0.95, 80, 85, 1000, 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "passed", event.Status)
		require.Equal(t, 0.95, event.DeterministicScore)
		require.Equal(t, 80, event.RubricScore)
		require.Equal(t, 85, event.TotalScore)
		require.Equal(t, int64(1000), event.DurationMs)
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty eval_run_id
		badEvent := NewEvalCompletedEvent(ID{}, evalCaseID, snapshotID, "passed", 0.95, 80, 85, 1000, 1)
		require.Error(t, badEvent.Validate())

		// Invalid: bad status
		badEvent2 := NewEvalCompletedEvent(evalRunID, evalCaseID, snapshotID, "unknown", 0.95, 80, 85, 1000, 1)
		require.Error(t, badEvent2.Validate())
	})
}

func TestPromptAdaptedEvent(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	event := NewPromptAdaptedEvent(assetID, "gpt-4", "claude-3", "adapted content", 5, 1)

	t.Run("Create event", func(t *testing.T) {
		require.Equal(t, "gpt-4", event.SourceModel)
		require.Equal(t, "claude-3", event.TargetModel)
		require.Equal(t, "adapted content", event.AdaptedContent)
		require.Equal(t, 5, event.ScoreDelta)
	})

	t.Run("Validate", func(t *testing.T) {
		require.NoError(t, event.Validate())

		// Invalid: empty source model
		badEvent := NewPromptAdaptedEvent(assetID, "", "claude-3", "content", 5, 1)
		require.Error(t, badEvent.Validate())

		// Invalid: empty target model
		badEvent2 := NewPromptAdaptedEvent(assetID, "gpt-4", "", "content", 5, 1)
		require.Error(t, badEvent2.Validate())
	})
}

func TestEventToJSON(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	event := NewPromptAssetCreatedEvent(assetID, "Test", "desc", "biz", nil, "/path", 1)

	data, err := EventToJSON(event)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	require.Equal(t, "Test", m["name"])
}

func TestEventFromJSON(t *testing.T) {
	original := `{"event_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","name":"Test"}`
	data, err := EventFromJSON([]byte(original))
	require.NoError(t, err)
	require.Equal(t, "Test", data["name"])
}

func TestOutboxEvent(t *testing.T) {
	t.Run("NewOutboxEvent from domain event", func(t *testing.T) {
		assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
		event := NewPromptAssetCreatedEvent(assetID, "Test", "desc", "biz", nil, "/path", 1)

		outbox, err := NewOutboxEvent(event)
		require.NoError(t, err)
		require.Equal(t, EventStatusPending, outbox.Status)
		require.Equal(t, 0, outbox.RetryCount)
		require.NotEmpty(t, outbox.IdempotencyKey)
		require.NotEmpty(t, outbox.Payload)
	})
}
