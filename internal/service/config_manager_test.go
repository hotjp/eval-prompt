package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryConfigManager_RegisterNotify(t *testing.T) {
	m := NewInMemoryConfigManager()

	called := atomic.Bool{}
	m.Register("test", func(ctx context.Context, domain string, changed []string) {
		called.Store(true)
		assert.Equal(t, "test", domain)
		assert.Equal(t, []string{"key1", "key2"}, changed)
	})

	m.Notify(context.Background(), "test", []string{"key1", "key2"})
	assert.True(t, called.Load(), "handler should have been called")
}

func TestInMemoryConfigManager_MultipleHandlers(t *testing.T) {
	m := NewInMemoryConfigManager()

	var count atomic.Int32
	m.Register("test", func(ctx context.Context, domain string, changed []string) {
		count.Add(1)
	})
	m.Register("test", func(ctx context.Context, domain string, changed []string) {
		count.Add(1)
	})
	m.Register("test", func(ctx context.Context, domain string, changed []string) {
		count.Add(1)
	})

	m.Notify(context.Background(), "test", nil)
	assert.Equal(t, int32(3), count.Load(), "all 3 handlers should be called")
}

func TestInMemoryConfigManager_DifferentDomains(t *testing.T) {
	m := NewInMemoryConfigManager()

	var llmCalled, repoCalled atomic.Bool
	m.Register("llm", func(ctx context.Context, domain string, changed []string) {
		llmCalled.Store(true)
	})
	m.Register("repo", func(ctx context.Context, domain string, changed []string) {
		repoCalled.Store(true)
	})

	m.Notify(context.Background(), "llm", nil)
	assert.True(t, llmCalled.Load())
	assert.False(t, repoCalled.Load())

	llmCalled.Store(false)
	m.Notify(context.Background(), "repo", nil)
	assert.False(t, llmCalled.Load())
	assert.True(t, repoCalled.Load())
}

func TestInMemoryConfigManager_Concurrent(t *testing.T) {
	m := NewInMemoryConfigManager()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Register("domain", func(ctx context.Context, domain string, changed []string) {
				// Simple handler that does nothing
			})
		}(i)
	}
	wg.Wait()

	var notifyWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		notifyWg.Add(1)
		go func() {
			defer notifyWg.Done()
			m.Notify(context.Background(), "domain", nil)
		}()
	}
	notifyWg.Wait()
	// If we get here without deadlock, the implementation is concurrency-safe
}

func TestInMemoryConfigManager_NoHandler(t *testing.T) {
	m := NewInMemoryConfigManager()
	// Should not panic when notifying a domain with no handlers
	m.Notify(context.Background(), "nonexistent", []string{"key"})
}
