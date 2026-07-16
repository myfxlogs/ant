package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"

	connectrpc "connectrpc.com/connect"
)

// registerShareHandlers wires the Share ConnectRPC handler.
func registerShareHandlers(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	log *zap.Logger,
	tradeRecordRepo *repository.TradeRecordRepository,
	userRepo *repository.UserRepository,
	mthubSvc *mthub.MtHubService,
	jwtSecret string,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	shareRepo := repository.NewShareRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	shareServer := user.NewShareServer(shareRepo, tradeRecordRepo, analyticsRepo, userRepo, mthubSvc, pool, jwtSecret, log)
	mux.Handle(antv1c.NewShareServiceHandler(shareServer, withSency(otelInterceptor, authInterceptor)))
}
