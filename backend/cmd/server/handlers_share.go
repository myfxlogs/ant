package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect/user"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
	"anttrader/internal/service"

	connectrpc "connectrpc.com/connect"
)

// registerShareHandlers wires the Share ConnectRPC handler + REST endpoints.
func registerShareHandlers(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	log *zap.Logger,
	tradeRecordRepo *repository.TradeRecordRepository,
	userRepo *repository.UserRepository,
	mthubSvc *mthub.MtHubService,
	platformSvc *service.PlatformService,
	jwtSecret string,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
	authInt *interceptor.AuthInterceptor,
) {
	shareRepo := repository.NewShareRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	shareServer := user.NewShareServer(shareRepo, tradeRecordRepo, analyticsRepo, userRepo, mthubSvc, pool, jwtSecret, log)
	mux.Handle(antv1c.NewShareServiceHandler(shareServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))

	mux.HandleFunc("/api/shares/create", shareServer.HandleCreateShareTokenREST)
	mux.HandleFunc("/api/shares/update", shareServer.HandleUpdateShareToken)
	mux.HandleFunc("/api/share/performance", shareServer.HandleGetSharedPerformanceJSON)
	mux.HandleFunc("/api/shares/delete", shareServer.HandleDeleteShareToken)
	mux.HandleFunc("/api/shares/list", shareServer.HandleListShareTokens)
	mux.HandleFunc("/api/admin/shares/list", func(w http.ResponseWriter, r *http.Request) {
		uid, err := authInt.UserIDFromHTTP(r)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ok, _ := platformSvc.IsAdmin(r.Context(), uid)
		if !ok {
			http.Error(w, `{"error":"admin required"}`, http.StatusForbidden)
			return
		}
		shareServer.HandleListAllShareTokens(w, r)
	})
}
