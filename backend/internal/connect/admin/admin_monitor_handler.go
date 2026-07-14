package admin

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mdgateway"
	antredis "alphaforge/internal/storage/redis"
)

// AdminMonitorServer streams real-time system metrics to admin UI via SSE.
type AdminMonitorServer struct {
	pool    *pgxpool.Pool
	rdb     *antredis.Client
	nc      *nats.Conn
	log     *zap.Logger
	startAt time.Time
}

var _ antv1c.AdminMonitorServiceHandler = (*AdminMonitorServer)(nil)

func NewAdminMonitorServer(pool *pgxpool.Pool, rdb *antredis.Client, nc *nats.Conn, log *zap.Logger) *AdminMonitorServer {
	return &AdminMonitorServer{
		pool:    pool,
		rdb:     rdb,
		nc:      nc,
		log:     log,
		startAt: time.Now(),
	}
}

func (s *AdminMonitorServer) SubscribeMetrics(
	ctx context.Context,
	_ *connect.Request[antv1.SubscribeMetricsRequest],
	stream *connect.ServerStream[antv1.MonitorSnapshot],
) error {
	uid := interceptor.GetUserID(ctx)
	if uid == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send immediately on connect.
	if err := stream.Send(s.collect(ctx)); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send monitor snapshot: %w", err))
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := stream.Send(s.collect(ctx)); err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("send monitor snapshot: %w", err))
			}
		}
	}
}

func (s *AdminMonitorServer) collect(ctx context.Context) *antv1.MonitorSnapshot {
	snap := &antv1.MonitorSnapshot{
		Timestamp:      timestamppb.Now(),
		UptimeSeconds:  int64(time.Since(s.startAt).Seconds()),
	}

	// Go runtime
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	snap.Goroutines = uint32(runtime.NumGoroutine())
	snap.HeapAllocBytes = memStats.HeapAlloc
	snap.HeapSysBytes = memStats.HeapSys
	snap.StackInuseBytes = memStats.StackInuse
	snap.NumGc = uint64(memStats.NumGC)
	if memStats.NumGC > 0 {
		snap.GcPauseAvgNs = float64(memStats.PauseTotalNs) / float64(memStats.NumGC)
	}

	// DB pool stats
	if s.pool != nil {
		stat := s.pool.Stat()
		snap.DbPoolTotal = int32(stat.TotalConns())
		snap.DbPoolIdle = int32(stat.IdleConns())
		snap.DbPoolAcquired = int32(stat.AcquiredConns())
		if err := s.pool.Ping(ctx); err != nil {
			snap.DbStatus = "unhealthy: " + err.Error()
		} else {
			snap.DbStatus = "ok"
		}
	} else {
		snap.DbStatus = "not configured"
	}

	// Redis
	if s.rdb != nil {
		if err := s.rdb.Client().Ping(ctx).Err(); err != nil {
			snap.RedisStatus = "unhealthy: " + err.Error()
		} else {
			snap.RedisStatus = "ok"
		}
	} else {
		snap.RedisStatus = "not configured"
	}

	// NATS
	if s.nc != nil {
		if s.nc.IsConnected() {
			snap.NatsStatus = "ok"
		} else {
			snap.NatsStatus = "disconnected"
		}
	} else {
		snap.NatsStatus = "not configured"
	}

	// mdgateway metrics
	snap.SpillPendingFiles = mdgateway.SpillPendingFilesCount()
	snap.DlqParseErrors = mdgateway.DLQSampled("parse_error")
	snap.DlqBidGtAsk = mdgateway.DLQSampled("bid_gt_ask")
	snap.DlqNonPositive = mdgateway.DLQSampled("non_positive")
	snap.BarDroppedTotal = mdgateway.BarSkippedFinalized()
	snap.SignalDroppedTotal = mdgateway.SignalDroppedTotal()
	snap.StaleAccounts = mdgateway.StaleAccountCount()
	snap.DeadAccounts = mdgateway.DeadAccountCount()
	snap.MdGapAvgSeconds = mdgateway.GapAvgSeconds()
	snap.MdGapMaxSeconds = mdgateway.GapMaxSeconds()
	snap.ConsumerLag = mdgateway.ConsumerLag()

	// Disk usage (root partition)
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		snap.DiskTotalBytes = stat.Blocks * uint64(stat.Bsize)
		snap.DiskUsedBytes = (stat.Blocks - stat.Bavail) * uint64(stat.Bsize)
		if snap.DiskTotalBytes > 0 {
			snap.DiskUsagePct = float64(snap.DiskUsedBytes) / float64(snap.DiskTotalBytes) * 100
		}
	}

	return snap
}

// FormatBytes formats byte count as human-readable string.
func FormatBytes(b uint64) string {
	d := decimal.NewFromInt(int64(b))
	switch {
	case b >= 1<<30:
		return d.Div(decimal.NewFromInt(1 << 30)).StringFixed(2) + " GB"
	case b >= 1<<20:
		return d.Div(decimal.NewFromInt(1 << 20)).StringFixed(2) + " MB"
	case b >= 1<<10:
		return d.Div(decimal.NewFromInt(1 << 10)).StringFixed(2) + " KB"
	default:
		return fmt.Sprintf("%d B", b)
	}
}
