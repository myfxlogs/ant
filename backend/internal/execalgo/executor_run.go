// Package execalgo — Executor runtime (execution loop + helpers).
package execalgo

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/internal/mthub"
)

// run is the main execution loop.
func (e *Executor) run(ctx context.Context) {
	defer close(e.events)

	slices := e.cfg.Schedule.Slices
	total := len(slices)

	e.emit(ExecEvent{State: ExecRunning, SliceIndex: -1, TotalSlices: total, Timestamp: time.Now()})

	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			e.transitionTo(ExecCancelled)
			return
		default:
		}

		e.mu.Lock()
		if e.state == ExecCancelled || e.state == ExecFailed {
			e.mu.Unlock()
			return
		}
		e.mu.Unlock()

		slice := slices[i]
		now := time.Now()

		// Wait until TargetTime.
		if slice.TargetTime.After(now) {
			waitDur := slice.TargetTime.Sub(now)
			timer := time.NewTimer(waitDur)
			select {
			case <-ctx.Done():
				timer.Stop()
				e.transitionTo(ExecCancelled)
				return
			case <-timer.C:
			}
		}

		// Market state check with retry.
		if e.cfg.MarketState != nil {
			tradeable, reason := e.cfg.MarketState.IsTradeable(e.cfg.Schedule.Parent.Symbol)
			if !tradeable {
				e.emit(ExecEvent{
					State: ExecPaused, SliceIndex: i, TotalSlices: total,
					Error: fmt.Errorf("market not tradeable: %s", reason), Timestamp: time.Now(),
				})
				retryTicker := time.NewTicker(5 * time.Second)
				defer retryTicker.Stop()
				for !tradeable {
					select {
					case <-ctx.Done():
						e.transitionTo(ExecCancelled)
						return
					case <-retryTicker.C:
						tradeable, reason = e.cfg.MarketState.IsTradeable(e.cfg.Schedule.Parent.Symbol)
					}
				}
				e.emit(ExecEvent{State: ExecRunning, SliceIndex: i, TotalSlices: total, Timestamp: time.Now()})
			}
		}

		// Submit child order.
		req := &mthub.OrderRequest{
			AccountID: e.cfg.AccountID,
			Canonical: e.cfg.Schedule.Parent.Symbol,
			Side:      sideToMthub(e.cfg.Schedule.Parent.Side),
			OrderType: mthub.OrderMarket,
			Volume:    decimal.NewFromFloat(slice.Volume),
		}
		if slice.LimitPrice > 0 {
			req.OrderType = mthub.OrderLimit
			req.Price = decimal.NewFromFloat(slice.LimitPrice)
		}

		ticket, err := e.cfg.Broker.SubmitOrder(ctx, req)
		if err != nil {
			e.mu.Lock()
			e.failedCount++
			e.mu.Unlock()
			e.emit(ExecEvent{
				State: ExecRunning, SliceIndex: i, TotalSlices: total,
				Error: fmt.Errorf("slice %d submit: %w", i, err), Timestamp: time.Now(),
			})
			continue
		}

		e.mu.Lock()
		e.submitted++
		e.nextSlice = i + 1
		e.mu.Unlock()

		e.emit(ExecEvent{
			State: ExecRunning, SliceIndex: i, TotalSlices: total,
			Ticket: ticket, Timestamp: time.Now(),
		})
	}

	e.transitionTo(ExecCompleted)
}

// transitionTo sets the state and emits an event.
func (e *Executor) transitionTo(s ExecState) {
	e.mu.Lock()
	e.state = s
	e.mu.Unlock()
	e.emit(ExecEvent{State: s, SliceIndex: -1, TotalSlices: len(e.cfg.Schedule.Slices), Timestamp: time.Now()})
}

// emit sends an event to the channel. Non-blocking: if full, the event is dropped.
func (e *Executor) emit(ev ExecEvent) {
	select {
	case e.events <- ev:
	default:
	}
}

// sideToMthub converts execalgo side string to mthub.Side.
func sideToMthub(side string) mthub.Side {
	if side == "sell" {
		return mthub.SideSell
	}
	return mthub.SideBuy
}
