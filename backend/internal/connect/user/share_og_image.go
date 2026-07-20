package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// ogImageServer serves dynamically generated SVG OG images for share tokens.
// Social media crawlers (Facebook, Twitter, LinkedIn) fetch OG images via HTTP GET,
// so this must be a plain HTTP endpoint — ConnectRPC cannot serve this use case.
type ogImageServer struct {
	repo         *repository.ShareRepository
	tradeRecords *repository.TradeRecordRepository
	eqRepo       *repository.AnalyticsRepository
	userRepo     *repository.UserRepository
	pg           *pgxpool.Pool
	log          *zap.Logger
}

func newOGImageServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, pg *pgxpool.Pool, log *zap.Logger) *ogImageServer {
	return &ogImageServer{repo: repo, tradeRecords: tradeRecords, eqRepo: eqRepo, userRepo: userRepo, pg: pg, log: log}
}

// NewOGImageServer creates an HTTP handler for /share/{token}/og-image.
func NewOGImageServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, pg *pgxpool.Pool, log *zap.Logger) http.Handler {
	return newOGImageServer(repo, tradeRecords, eqRepo, userRepo, pg, log)
}

func (s *ogImageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract token from path: /share/{token}/og-image
	path := r.URL.Path
	if !strings.HasSuffix(path, "/og-image") {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimPrefix(path, "/share/")
	token = strings.TrimSuffix(token, "/og-image")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	st, err := s.repo.GetByToken(ctx, token)
	if err != nil || st == nil || time.Now().After(st.ExpiresAt) {
		http.NotFound(w, r)
		return
	}

	user, _ := s.userRepo.GetByID(ctx, st.UserID)
	userName := "Anonymous"
	if user != nil {
		if user.Nickname != nil && *user.Nickname != "" {
			userName = *user.Nickname
		} else if user.Email != "" {
			userName = user.Email
		}
	}

	// Fetch performance data (reuse same logic as GetSharedPerformance).
	aid, _ := uuid.Parse(st.AccountID)
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()
	equityPoints, _ := s.eqRepo.GetEquityCurve(ctx, aid, start, end)

	trades, _ := s.tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	stats := summarizeTrades(trades)

	// Build SVG.
	svg := renderOGImageSVG(userName, stats.totalReturnStr(), stats.winRateStr(), stats.maxDrawdownStr(), fmt.Sprintf("%d", len(trades)), fmt.Sprintf("%.4f", computeSharpe(equityPoints)))

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(svg))
}

// computeSharpe calculates annualized Sharpe ratio from equity curve points.
func computeSharpe(equityPoints []*model.EquityPoint) float64 {
	if len(equityPoints) < 2 {
		return 0
	}
	var sum, sumSq float64
	var returns []float64
	for i := 1; i < len(equityPoints); i++ {
		prev, ok1 := equityPoints[i-1].Equity.Float64()
		if !ok1 || prev == 0 {
			continue
		}
		curr, ok2 := equityPoints[i].Equity.Float64()
		if !ok2 {
			continue
		}
		r := (curr - prev) / prev
		returns = append(returns, r)
		sum += r
	}
	if len(returns) < 2 {
		return 0
	}
	n := float64(len(returns))
	mean := sum / n
	for _, r := range returns {
		diff := r - mean
		sumSq += diff * diff
	}
	variance := sumSq / (n - 1)
	if variance <= 0 {
		return 0
	}
	std := sqrt(variance)
	if std == 0 {
		return 0
	}
	return mean / std * sqrt(252)
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method — avoid math import dependency issues
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// renderOGImageSVG generates a 1200×630 SVG with performance metrics overlaid.
func renderOGImageSVG(userName, totalReturn, winRate, maxDrawdown, totalTrades, sharpe string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <defs>
    <linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#0f1923"/>
      <stop offset="100%%" style="stop-color:#1a2a3a"/>
    </linearGradient>
    <linearGradient id="accent" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
      <stop offset="0%%" style="stop-color:#d4af37"/>
      <stop offset="100%%" style="stop-color:#b8960b"/>
    </linearGradient>
  </defs>
  <rect width="1200" height="630" fill="url(#bg)"/>
  <rect x="0" y="0" width="1200" height="6" fill="url(#accent)"/>

  <!-- Brand -->
  <text x="60" y="80" font-family="Segoe UI, Helvetica, sans-serif" font-size="28" font-weight="700" fill="#d4af37">AlphaForge</text>
  <text x="60" y="110" font-family="Segoe UI, Helvetica, sans-serif" font-size="16" fill="#8c8c8c">Trading Performance Report</text>

  <!-- User name -->
  <text x="60" y="180" font-family="Segoe UI, Helvetica, sans-serif" font-size="36" font-weight="600" fill="#ffffff">%s</text>

  <!-- Metrics grid -->
  <g font-family="Segoe UI, Helvetica, sans-serif">
    <!-- Total Return -->
    <text x="60" y="280" font-size="14" fill="#8c8c8c">Total Return</text>
    <text x="60" y="320" font-size="42" font-weight="700" fill="#52c41a">%s</text>

    <!-- Win Rate -->
    <text x="340" y="280" font-size="14" fill="#8c8c8c">Win Rate</text>
    <text x="340" y="320" font-size="42" font-weight="700" fill="#ffffff">%s%%</text>

    <!-- Max Drawdown -->
    <text x="620" y="280" font-size="14" fill="#8c8c8c">Max Drawdown</text>
    <text x="620" y="320" font-size="42" font-weight="700" fill="#ff4d4f">%s</text>

    <!-- Total Trades -->
    <text x="900" y="280" font-size="14" fill="#8c8c8c">Total Trades</text>
    <text x="900" y="320" font-size="42" font-weight="700" fill="#ffffff">%s</text>

    <!-- Sharpe Ratio -->
    <text x="60" y="430" font-size="14" fill="#8c8c8c">Sharpe Ratio</text>
    <text x="60" y="470" font-size="36" font-weight="600" fill="#1890ff">%s</text>
  </g>

  <!-- Footer -->
  <rect x="0" y="580" width="1200" height="50" fill="#0a1018"/>
  <text x="60" y="612" font-family="Segoe UI, Helvetica, sans-serif" font-size="14" fill="#5c6b7a">Verified on AlphaForge — alfq.org</text>
</svg>`, userName, totalReturn, winRate, maxDrawdown, totalTrades, sharpe)
}
