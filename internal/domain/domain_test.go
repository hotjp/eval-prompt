package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateRoot_RecordEvent(t *testing.T) {
	root := &AggregateRoot{}
	require.Empty(t, root.events)

	event := NewPromptAssetCreatedEvent(ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, "Test", "desc", "biz", nil, "/path", 1)
	root.RecordEvent(event)

	require.Len(t, root.events, 1)
	require.Equal(t, event, root.events[0])
}

func TestAggregateRoot_RecordMultipleEvents(t *testing.T) {
	root := &AggregateRoot{}
	event1 := NewPromptAssetCreatedEvent(ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, "Test1", "desc", "biz", nil, "/path", 1)
	event2 := NewPromptAssetUpdatedEvent(ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, "hash", "reason", 2)

	root.RecordEvent(event1)
	root.RecordEvent(event2)

	require.Len(t, root.events, 2)
}

func TestAggregateRoot_FlushEvents(t *testing.T) {
	root := &AggregateRoot{}
	event := NewPromptAssetCreatedEvent(ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, "Test", "desc", "biz", nil, "/path", 1)
	root.RecordEvent(event)

	flushed := root.FlushEvents()

	require.Len(t, flushed, 1)
	require.Empty(t, root.events)
}

func TestAggregateRoot_FlushEmpty(t *testing.T) {
	root := &AggregateRoot{}
	flushed := root.FlushEvents()
	require.Empty(t, flushed)
}

func TestNewAggregateRoot(t *testing.T) {
	root := NewAggregateRoot()

	require.NotEmpty(t, root.ID.String())
	require.Equal(t, int64(0), root.Version)
	require.Empty(t, root.events)
}

func TestAggregateRoot_IncrementVersion(t *testing.T) {
	root := &AggregateRoot{Entity: Entity{Version: 0}}

	root.IncrementVersion()
	require.Equal(t, int64(1), root.Version)

	root.IncrementVersion()
	require.Equal(t, int64(2), root.Version)
}
