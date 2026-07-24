package user

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// ogImageServer serves dynamically generated PNG OG images for share tokens.
// Social media crawlers (Facebook, Twitter, LinkedIn, Slack) fetch OG images via
// HTTP GET and require PNG/JPG — SVG is NOT supported by most crawlers.
// This must be a plain HTTP endpoint — ConnectRPC cannot serve this use case.
type ogImageServer struct {
	repo         *repository.ShareRepository
	tradeRecords *repository.TradeRecordRepository
	eqRepo       *repository.AnalyticsRepository
	userRepo     *repository.UserRepository
	log          *zap.Logger
}

func newOGImageServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, log *zap.Logger) *ogImageServer {
	return &ogImageServer{repo: repo, tradeRecords: tradeRecords, eqRepo: eqRepo, userRepo: userRepo, log: log}
}

// NewOGImageServer creates an HTTP handler for /share/{token}/og-image.
func NewOGImageServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, log *zap.Logger) http.Handler {
	return newOGImageServer(repo, tradeRecords, eqRepo, userRepo, log)
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
	if err != nil {
		s.log.Warn("og-image: GetByToken", zap.String("token", token), zap.Error(err))
		http.NotFound(w, r)
		return
	}
	if st == nil || time.Now().After(st.ExpiresAt) {
		http.NotFound(w, r)
		return
	}

	user, err := s.userRepo.GetByID(ctx, st.UserID)
	if err != nil {
		s.log.Warn("og-image: GetByID", zap.String("user_id", st.UserID.String()), zap.Error(err))
	}
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
	equityPoints, err := s.eqRepo.GetEquityCurve(ctx, aid, start, end)
	if err != nil {
		s.log.Warn("og-image: GetEquityCurve", zap.String("account_id", st.AccountID), zap.Error(err))
	}

	trades, err := s.tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	if err != nil {
		s.log.Warn("og-image: GetByAccountID", zap.String("account_id", st.AccountID), zap.Error(err))
	}
	stats := summarizeTrades(trades)

	// Build PNG.
	pngData := renderOGImagePNG(userName, stats.totalReturnStr(), stats.winRateStr(), stats.maxDrawdownStr(), fmt.Sprintf("%d", len(trades)), fmt.Sprintf("%.4f", computeSharpe(equityPoints)))

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(pngData)
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
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}
	return mean / std * math.Sqrt(252)
}

// renderOGImagePNG generates a 1200×630 PNG with performance metrics.
// Uses Go standard image package + basicfont for text rendering.
func renderOGImagePNG(userName, totalReturn, winRate, maxDrawdown, totalTrades, sharpe string) []byte {
	const (
		W = 1200
		H = 630
	)
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Background: dark navy (#0f1923)
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 0x0f, G: 0x19, B: 0x23, A: 0xff}}, image.Point{}, draw.Src)

	// Top accent bar: gold (#d4af37)
	draw.Draw(img, image.Rect(0, 0, W, 6), &image.Uniform{color.RGBA{R: 0xd4, G: 0xaf, B: 0x37, A: 0xff}}, image.Point{}, draw.Src)

	// Footer bar: darker (#0a1018)
	draw.Draw(img, image.Rect(0, 580, W, H), &image.Uniform{color.RGBA{R: 0x0a, G: 0x10, B: 0x18, A: 0xff}}, image.Point{}, draw.Src)

	gold := color.RGBA{R: 0xd4, G: 0xaf, B: 0x37, A: 0xff}
	white := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	gray := color.RGBA{R: 0x8c, G: 0x8c, B: 0x8c, A: 0xff}
	green := color.RGBA{R: 0x52, G: 0xc4, B: 0x1a, A: 0xff}
	red := color.RGBA{R: 0xff, G: 0x4d, B: 0x4f, A: 0xff}
	blue := color.RGBA{R: 0x18, G: 0x90, B: 0xff, A: 0xff}
	footerGray := color.RGBA{R: 0x5c, G: 0x6b, B: 0x7a, A: 0xff}

	drawText(img, 60, 70, "AlphaForge", gold, 2)
	drawText(img, 60, 100, "Trading Performance Report", gray, 1)
	drawTextBig(img, 60, 160, userName, white)

	// Metrics row 1 (y=250 label, y=290 value)
	drawText(img, 60, 250, "Total Return", gray, 1)
	drawTextLarge(img, 60, 300, totalReturn, green)

	drawText(img, 340, 250, "Win Rate", gray, 1)
	drawTextLarge(img, 340, 300, winRate+"%", white)

	drawText(img, 620, 250, "Max Drawdown", gray, 1)
	drawTextLarge(img, 620, 300, maxDrawdown, red)

	drawText(img, 900, 250, "Total Trades", gray, 1)
	drawTextLarge(img, 900, 300, totalTrades, white)

	// Metrics row 2 (y=400 label, y=440 value)
	drawText(img, 60, 400, "Sharpe Ratio", gray, 1)
	drawTextLarge(img, 60, 450, sharpe, blue)

	// Footer text
	drawText(img, 60, 605, "Verified on AlphaForge — alfq.org", footerGray, 1)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// drawText renders text at (x, y) using basicfont with optional scale.
// scale=1 uses 8px font, scale=2 uses 16px (pixel-doubled), etc.
func drawText(img *image.RGBA, x, y int, text string, c color.Color, scale int) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	if scale == 1 {
		d.DrawString(text)
		return
	}
	// For scaled text, draw to a temp image then scale-blit.
	tw := len(text)*7 + 1
	th := 14
	tmp := image.NewRGBA(image.Rect(0, 0, tw, th))
	td := font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(0, 11),
	}
	td.DrawString(text)
	// Scale-blit: copy each pixel scale×scale.
	for ty := 0; ty < th; ty++ {
		for tx := 0; tx < tw; tx++ {
			pixel := tmp.At(tx, ty)
			_, _, _, a := pixel.RGBA()
			if a == 0 {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set(x+tx*scale+dx, y-11*scale+ty*scale+dy, pixel)
				}
			}
		}
	}
}

// drawTextBig renders text at 3x scale (21px effective) for headings.
func drawTextBig(img *image.RGBA, x, y int, text string, c color.Color) {
	drawText(img, x, y, text, c, 3)
}

// drawTextLarge renders text at 4x scale (28px effective) for metric values.
func drawTextLarge(img *image.RGBA, x, y int, text string, c color.Color) {
	drawText(img, x, y, text, c, 4)
}
