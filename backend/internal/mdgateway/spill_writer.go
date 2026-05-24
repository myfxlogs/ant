package mdgateway

import "anttrader/internal/mdgateway/adapter/mdtick"

type SpillWriter struct{}

func NewSpillWriter() *SpillWriter { return &SpillWriter{} }
func (w *SpillWriter) WriteTick(t *mdtick.Tick) error { return nil }
func (w *SpillWriter) WriteBar(b *mdtick.Bar) error { return nil }
