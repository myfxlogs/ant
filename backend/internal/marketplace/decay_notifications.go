package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// notifyDecayDetected sends notifications to the strategy author and all active
// subscribers when decay is detected. Author gets "want to iterate?" prompt;
// buyers get "your subscribed strategy is decaying" alert.
// Does NOT auto-create optimization tasks — author must manually initiate.
func (s *Service) notifyDecayDetected(ctx context.Context, result *DecayResult, status string) {
	if s.notifSender == nil {
		return
	}

	sid, err := uuid.Parse(result.StrategyID)
	if err != nil {
		return
	}

	// Fetch strategy title and publisher.
	var title string
	var publisherID uuid.UUID
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(title, ''), publisher_id
		 FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&title, &publisherID)
	if err != nil {
		return
	}

	// Notify author: "Your strategy is decaying, want to iterate?"
	authorMsg := fmt.Sprintf(
		"Your strategy \"%s\" is showing alpha decay (%s). %s. Consider iterating to improve performance.",
		title, status, result.TriggerReason,
	)
	_, _ = s.notifSender.Send(ctx, publisherID, "decay_detected",
		"Strategy Decay Detected", authorMsg, nil)

	// Notify all active subscribers: "Your subscribed strategy is decaying"
	rows, err := s.pg.Query(ctx,
		`SELECT subscriber_user_id FROM user_subscriptions
		 WHERE target_strategy_id = $1 AND active = true`,
		sid)
	if err != nil {
		return
	}
	defer rows.Close()

	buyerMsg := fmt.Sprintf(
		"A strategy you subscribed to (\"%s\") is showing performance decay. Review your subscription.",
		title,
	)
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, _ = s.notifSender.Send(ctx, uid, "subscribed_strategy_decay",
			"Subscribed Strategy Decay", buyerMsg, nil)
	}
}
