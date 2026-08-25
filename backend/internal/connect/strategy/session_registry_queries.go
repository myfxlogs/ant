// session_registry_queries.go — Query and update methods extracted from session_registry.go.
package strategy

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

func (r *SessionRegistry) UpdatePnlFromPositions(accountID string, positions []mdtick.ProfitPosition) {
	r.mu.RLock()
	byMagic := make(map[int32]*ActiveSession)
	for _, sess := range r.sessions {
		if sess.AccountID == accountID {
			byMagic[sess.MagicNumber] = sess
		}
	}
	r.mu.RUnlock()
	if len(byMagic) == 0 {
		return
	}

	// Sum profit for each magic number. ProfitPosition already contains the
	// running profit for that position, so simple attribution-by-magic works.
	pnlByMagic := make(map[int32]decimal.Decimal)
	for _, pos := range positions {
		if pos.Magic == 0 {
			continue
		}
		pnlByMagic[pos.Magic] = pnlByMagic[pos.Magic].Add(pos.Profit)
	}

	// Update PnL for all active sessions on this account. Sessions whose magic
	// number has no open position get PnL reset to "0" (position closed = no PnL).
	for magic, sess := range byMagic {
		if pnl, ok := pnlByMagic[magic]; ok {
			sess.SetPnL(pnl.String())
		} else {
			sess.SetPnL("0")
		}
	}
}

// ListByAccount returns all active sessions for an account.
func (r *SessionRegistry) ListByAccount(accountID string) []*ActiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ActiveSession
	for _, sess := range r.sessions {
		if sess.AccountID == accountID {
			out = append(out, sess)
		}
	}
	return out
}

// ListAll returns all active sessions.
func (r *SessionRegistry) ListAll() []*ActiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ActiveSession, 0, len(r.sessions))
	for _, sess := range r.sessions {
		out = append(out, sess)
	}
	return out
}

// Stop cancels a session's context, causing RunLiveStrategy to exit.
func (r *SessionRegistry) Stop(runID uuid.UUID) error {
	r.mu.RLock()
	sess, ok := r.sessions[runID]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", runID)
	}
	sess.cancel()
	return nil
}

// RecordTick updates the latest tick timestamp for this session.
