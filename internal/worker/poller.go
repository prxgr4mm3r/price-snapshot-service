package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/internal/ports"
)

// Poller polls prices at regular intervals
type Poller struct {
	service  ports.PollerService
	interval time.Duration
	logger   *slog.Logger

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewPoller creates a new price poller
func NewPoller(service ports.PollerService, interval time.Duration, logger *slog.Logger) *Poller {
	return &Poller{
		service:  service,
		interval: interval,
		logger:   logger.With("component", "poller"),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins polling prices
func (p *Poller) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	p.mu.Unlock()

	p.logger.Info("starting poller", "interval", p.interval.String())

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Initial poll
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poller context cancelled")
			close(p.doneCh)
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			return ctx.Err()

		case <-p.stopCh:
			p.logger.Info("poller stopped")
			close(p.doneCh)
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			return nil

		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	// Create a context with timeout for this poll
	pollTimeout := p.interval / 2
	if pollTimeout < 5*time.Second {
		pollTimeout = 5 * time.Second
	}

	pollCtx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()

	if err := p.service.PollPrices(pollCtx); err != nil {
		p.logger.Error("poll failed", "error", err)
	}
}

// Stop gracefully stops the poller
func (p *Poller) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	p.logger.Info("stopping poller")
	close(p.stopCh)

	// Wait for poller to finish with timeout
	select {
	case <-p.doneCh:
		return nil
	case <-time.After(10 * time.Second):
		return context.DeadlineExceeded
	}
}

// IsRunning returns whether the poller is currently running
func (p *Poller) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}
