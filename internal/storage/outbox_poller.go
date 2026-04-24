package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/eval-prompt/internal/storage/ent/outboxevent"
)

// OutboxPoller polls the outbox table and processes pending events.
type OutboxPoller struct {
	client *Client
	ticker *time.Ticker
	done   chan struct{}
	logger *slog.Logger
}

// NewOutboxPoller creates a new OutboxPoller.
func NewOutboxPoller(client *Client, logger *slog.Logger) *OutboxPoller {
	return &OutboxPoller{
		client: client,
		done:   make(chan struct{}),
		logger: logger,
	}
}

// Start begins polling the outbox table every 5 seconds.
func (p *OutboxPoller) Start(ctx context.Context) {
	p.ticker = time.NewTicker(5 * time.Second)
	p.logger.Info("outbox poller started", "interval", "5s")

	go func() {
		for {
			select {
			case <-ctx.Done():
				p.stop()
				return
			case <-p.done:
				return
			case <-p.ticker.C:
				if err := p.processOutbox(ctx); err != nil {
					p.logger.Error("failed to process outbox", "error", err)
				}
			}
		}
	}()
}

// Stop stops the outbox poller.
func (p *OutboxPoller) stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)
	p.logger.Info("outbox poller stopped")
}

// processOutbox processes pending outbox events.
func (p *OutboxPoller) processOutbox(ctx context.Context) error {
	// Start a transaction
	tx, err := p.client.ent.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Query pending events
	events, err := tx.OutboxEvent.Query().
		Where(outboxevent.StatusEQ(outboxevent.StatusPending)).
		Limit(100).
		All(ctx)

	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	p.logger.Debug("processing outbox events", "count", len(events))

	// Process each event
	for _, event := range events {
		// In a full implementation, we would:
		// 1. Send the event to the appropriate handler (Redis, webhook, etc.)
		// 2. If successful, mark as processed
		// 3. If failed, increment retry count and possibly mark as failed

		// For now, we just mark it as processed (local mode)
		_, err := tx.OutboxEvent.UpdateOne(event).
			SetStatus(outboxevent.StatusProcessed).
			Save(ctx)

		if err != nil {
			p.logger.Error("failed to mark event as processed",
				"event_id", event.ID,
				"error", err)
			return err
		}

		p.logger.Debug("processed outbox event",
			"event_id", event.ID,
			"event_type", event.EventType,
			"aggregate_id", event.AggregateID)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	p.logger.Info("outbox events processed", "count", len(events))
	return nil
}

// ProcessNow triggers an immediate processing of pending events.
func (p *OutboxPoller) ProcessNow(ctx context.Context) error {
	return p.processOutbox(ctx)
}
