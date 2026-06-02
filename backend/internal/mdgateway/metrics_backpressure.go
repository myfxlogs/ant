package mdgateway

import "sync/atomic"

// --- M10-BASE-B6: Backpressure metrics ---

var (
	chanFullTotal           atomic.Int64
	natsPublishDroppedTotal atomic.Int64
	consumerLag             atomic.Int64
	signalDroppedTotal      atomic.Int64
)

// RecordChanFull increments the bounded-channel-full counter.
func RecordChanFull() { chanFullTotal.Add(1) }

// ChanFullTotal returns the total count of channel-full drops.
func ChanFullTotal() int64 { return chanFullTotal.Load() }

// RecordNATSPublishDropped increments the NATS publish drop counter.
func RecordNATSPublishDropped() { natsPublishDroppedTotal.Add(1) }

// NATSPublishDroppedTotal returns the total NATS publish drops.
func NATSPublishDroppedTotal() int64 { return natsPublishDroppedTotal.Load() }

// SetConsumerLag sets the current consumer lag gauge (in messages).
func SetConsumerLag(lag int64) { consumerLag.Store(lag) }

// ConsumerLag returns the current consumer lag.
func ConsumerLag() int64 { return consumerLag.Load() }

// RecordSignalDropped increments the signal dropped counter.
func RecordSignalDropped() { signalDroppedTotal.Add(1) }

// SignalDroppedTotal returns the total dropped signals.
func SignalDroppedTotal() int64 { return signalDroppedTotal.Load() }
