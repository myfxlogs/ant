package mdgateway

import "anttrader/internal/mdgateway/adapter/mdtick"

type CHWriter struct{}

func NewCHWriter() *CHWriter { return &CHWriter{} }
func (w *CHWriter) EnqueueTick(t *mdtick.Tick) {}
func (w *CHWriter) EnqueueBar(b *mdtick.Bar) {}
