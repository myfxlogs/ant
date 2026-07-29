package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"alphaforge/internal/config"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
	antredis "alphaforge/internal/storage/redis"
	"alphaforge/internal/secrets"
	"alphaforge/internal/service"

	connectrpc "connectrpc.com/connect"
)

// handlerDeps holds all shared dependencies passed to registerHandlers.
type handlerDeps struct {
	Ctx               *context.Context
	Mux               *http.ServeMux
	Log               *zap.Logger
	Pool              *pgxpool.Pool
	Store             repository.MarketDataStore
	NC                *nats.Conn
	RDB               *antredis.Client
	Cfg               *config.Config
	JWTSecret         string
	AccountSvc        *service.AccountService
	PlatformSvc       *service.PlatformService
	AuthInterceptor   *interceptor.AuthInterceptor
	AdminInterceptor  *interceptor.AdminInterceptor
	RateLimitInterceptor *interceptor.RateLimitInterceptor
	OtelInterceptor   connectrpc.Interceptor
	MthubSvc          *mthub.MtHubService
	Hub               *mthub.Hub
	TradeRecordRepo   *repository.TradeRecordRepository
	JS                nats.JetStreamContext
	EventStore        *mthub.TradeEventStore
	ReconcileGate     *mthub.ReconcileGate
	AnalyticsCache    *service.AnalyticsCache
	BrokerReg         *adapter.BrokerRegistry
	SecClient         secrets.Client
	MktplaceSvc       *marketplace.Service
}

// interceptorSet groups the three interceptors used by admin handlers.
type interceptorSet struct {
	otel  connectrpc.Interceptor
	auth  connectrpc.Interceptor
	admin connectrpc.Interceptor
}
