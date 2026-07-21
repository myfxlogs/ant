package main

import (
	"context"
	"net/http"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	mktplace "alphaforge/internal/connect/marketplace"
	internalmkt "alphaforge/internal/marketplace"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"

	connectrpc "connectrpc.com/connect"
)

// registerMarketplaceHandlers wires market and marketplace ConnectRPC handlers.
// It returns the marketplace handler for later generator/streaming wiring.
func registerMarketplaceHandlers(
	ctx context.Context,
	mux *http.ServeMux,
	nc *nats.Conn,
	log *zap.Logger,
	store repository.MarketDataStore,
	mktplaceSvc *internalmkt.Service,
	walletRepo *repository.WalletRepository,
	platformSvc *service.PlatformService,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) *mktplace.MarketplaceServer {
	mktServer := mktplace.NewMarketServer(platformSvc, store, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, withSency(otelInterceptor, authInterceptor)))

	mktplaceSvc.SetWalletRepo(walletRepo)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, platformSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, withSency(otelInterceptor, authInterceptor)))

	mktplaceSvc.StartRenewalLoop(ctx, log)
	return mktplaceHandler
}
