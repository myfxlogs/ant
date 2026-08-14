package main

import (
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
)

// convertProfitPositions converts mdtick.ProfitPosition slice to mthub.AccountProfitPosition slice.
func convertProfitPositions(positions []mdtick.ProfitPosition) []mthub.AccountProfitPosition {
	out := make([]mthub.AccountProfitPosition, 0, len(positions))
	for _, pos := range positions {
		out = append(out, mthub.AccountProfitPosition{
			Ticket:       pos.Ticket,
			Symbol:       pos.Symbol,
			Magic:        pos.Magic,
			Profit:       pos.Profit,
			Volume:       pos.Volume,
			CurrentPrice: pos.CurrentPrice,
		})
	}
	return out
}
