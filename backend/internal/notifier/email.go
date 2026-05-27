// Package notifier provides push notification channels (email first).
package notifier

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// EmailConfig holds SMTP configuration.
type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	To       []string
}

// EmailNotifier sends alert emails via SMTP.
type EmailNotifier struct {
	cfg EmailConfig
	log *zap.Logger
}

// NewEmailNotifier creates an email notifier. Returns nil if SMTP is not configured.
func NewEmailNotifier(cfg EmailConfig, log *zap.Logger) *EmailNotifier {
	if cfg.Host == "" {
		log.Warn("SMTP not configured, email notifications disabled")
		return nil
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil {
		log.Warn("invalid SMTP_FROM address, email notifications disabled", zap.Error(err))
		return nil
	}
	return &EmailNotifier{cfg: cfg, log: log}
}

// Send sends an email with the given subject and plain-text body.
func (n *EmailNotifier) Send(subject, body string) error {
	if n == nil {
		return nil
	}

	to := make([]string, 0, len(n.cfg.To))
	for _, addr := range n.cfg.To {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			to = append(to, addr)
		}
	}
	if len(to) == 0 {
		n.log.Debug("no recipients configured, skipping email")
		return nil
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		n.cfg.From,
		strings.Join(to, ", "),
		subject,
		time.Now().Format(time.RFC1123Z),
		body,
	)

	addr := net.JoinHostPort(n.cfg.Host, n.cfg.Port)
	auth := smtp.PlainAuth("", n.cfg.User, n.cfg.Password, n.cfg.Host)

	// Try STARTTLS first; fall back to plain if not available.
	tlsConfig := &tls.Config{ServerName: n.cfg.Host}
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	if ok, _ := conn.Extension("STARTTLS"); ok {
		if err := conn.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if auth != nil {
		if err := conn.Auth(auth); err != nil {
			n.log.Warn("smtp auth failed, trying without auth", zap.Error(err))
		}
	}

	if err := conn.Mail(n.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := conn.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}
	wc, err := conn.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := fmt.Fprint(wc, msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	if err := conn.Quit(); err != nil {
		n.log.Debug("smtp quit", zap.Error(err))
	}

	n.log.Info("email sent", zap.String("subject", subject), zap.Strings("to", to))
	return nil
}

// --- Alert constructors for specific triggers ---

// MarginCallAlert sends a margin call notification.
func (n *EmailNotifier) MarginCallAlert(accountID, userID string, margin float64, equity float64) {
	subject := fmt.Sprintf("[ANT ALERT] Margin Call — account %s", accountID)
	body := fmt.Sprintf(
		"Account: %s\nUser: %s\nMargin: %.2f\nEquity: %.2f\nTime: %s\n\nAction required: deposit funds or reduce positions immediately.",
		accountID, userID, margin, equity, time.Now().Format(time.RFC3339),
	)
	if err := n.Send(subject, body); err != nil {
		n.log.Error("margin call email failed", zap.Error(err))
	}
}

// KillSwitchAlert sends a kill switch engagement notification.
func (n *EmailNotifier) KillSwitchAlert(reason, operator string) {
	subject := "[ANT ALERT] Kill Switch ENGAGED"
	body := fmt.Sprintf(
		"Kill Switch has been engaged.\nReason: %s\nOperator: %s\nTime: %s\n\nAll trading has been stopped.",
		reason, operator, time.Now().Format(time.RFC3339),
	)
	if err := n.Send(subject, body); err != nil {
		n.log.Error("kill switch email failed", zap.Error(err))
	}
}

// BreakerTripAlert sends a strategy breaker trip notification.
func (n *EmailNotifier) BreakerTripAlert(strategyID string, lossPct float64) {
	subject := fmt.Sprintf("[ANT ALERT] Strategy Breaker Tripped — %s", strategyID)
	body := fmt.Sprintf(
		"Strategy: %s\nLoss: %.2f%%\nTime: %s\n\nThe breaker has opened. Trades for this strategy are blocked until cooldown expires or manual reset.",
		strategyID, lossPct, time.Now().Format(time.RFC3339),
	)
	if err := n.Send(subject, body); err != nil {
		n.log.Error("breaker trip email failed", zap.Error(err))
	}
}

// PromoteToLiveAlert sends a strategy promotion notification.
func (n *EmailNotifier) PromoteToLiveAlert(strategyID, version string) {
	subject := fmt.Sprintf("[ANT INFO] Strategy Promoted to Live — %s", strategyID)
	body := fmt.Sprintf(
		"Strategy: %s\nVersion: %s\nTime: %s\n\nThe strategy has been promoted from canary/paper to live trading.",
		strategyID, version, time.Now().Format(time.RFC3339),
	)
	if err := n.Send(subject, body); err != nil {
		n.log.Error("promote to live email failed", zap.Error(err))
	}
}

// AnomalyAlert sends a generic anomaly/DDoS alert.
func (n *EmailNotifier) AnomalyAlert(detail string) {
	subject := "[ANT ALERT] Anomaly Detected"
	body := fmt.Sprintf(
		"Time: %s\n\n%s\n\nThis may indicate a DDoS attack or abnormal RPC traffic.",
		time.Now().Format(time.RFC3339), detail,
	)
	if err := n.Send(subject, body); err != nil {
		n.log.Error("anomaly email failed", zap.Error(err))
	}
}
