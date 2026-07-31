package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/sweep"

	connectrpc "connectrpc.com/connect"
)

// authHandlerParams holds parameters for registerAuthHandler.
type authHandlerParams struct {
	Mux                  *http.ServeMux
	Pool                 *pgxpool.Pool
	Cfg                  *config.Config
	JWTSecret            string
	UserRepo             *repository.UserRepository
	RegistrationSvc      *service.RegistrationService
	EmailNotifier        *notifier.EmailNotifier
	MTTester             user.MTConnectionTester
	Log                  *zap.Logger
	OtelInterceptor      connectrpc.Interceptor
	RateLimitInterceptor connectrpc.Interceptor
	AuthInterceptor      connectrpc.Interceptor
}

// registerAuthHandler wires the auth ConnectRPC handler and returns the AuthServer
// for modules that need to reference it (SRE, admin, etc).
func registerAuthHandler(p authHandlerParams) *user.AuthServer {
	mux := p.Mux
	pool := p.Pool
	cfg := p.Cfg
	log := p.Log
	authServer := user.NewAuthServer(p.UserRepo, p.JWTSecret, log)
	authServer.SetInsecureCookies(!cfg.CookieSecure)
	authServer.WithRegistration(p.RegistrationSvc)
	if p.EmailNotifier != nil {
		emailVerifSvc := service.NewEmailVerificationService(pool, p.EmailNotifier, cfg.AppURL, log)
		authServer.WithEmailVerification(emailVerifSvc)
		p.RegistrationSvc.SetEmailVerification(emailVerifSvc)
	}
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	authServer.WithPasswordReset(passwordResetRepo, p.EmailNotifier, cfg.AppURL)
	authServer.WithMTIdentityVerification(pool, p.MTTester)
	authServer.SetRequireEmailVerification(cfg.RequireEmailVerification)
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, withSency(p.OtelInterceptor, p.RateLimitInterceptor, p.AuthInterceptor)))
	return authServer
}

// registerWalletHandler wires the wallet ConnectRPC handler.
func registerWalletHandler(
	mux *http.ServeMux,
	walletSvc *service.WalletService,
	platformSvc *service.PlatformService,
	log *zap.Logger,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	walletServer := user.NewWalletServer(walletSvc, platformSvc, log)
	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, withSency(otelInterceptor, authInterceptor)))
}

// registerDepositHandler wires the deposit ConnectRPC handler.
func registerDepositHandler(
	mux *http.ServeMux,
	depositSvc *service.DepositService,
	platformSvc *service.PlatformService,
	sweepWorker *sweep.Worker,
	adminRepo *repository.AdminRepository,
	xpubFingerprint string,
	log *zap.Logger,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	depositServer := user.NewDepositServer(depositSvc, platformSvc, sweepWorker, adminRepo, xpubFingerprint, log)
	mux.Handle(antv1c.NewDepositServiceHandler(depositServer, withSency(otelInterceptor, authInterceptor)))
}

// registerAccountHandler wires the account ConnectRPC handler.
func registerAccountHandler(
	mux *http.ServeMux,
	cfg *config.Config,
	accountSvc *service.AccountService,
	accountEventPub *mdgateway.AccountEventPublisher,
	hub *mthub.Hub,
	mtTester user.MTConnectionTester,
	searcher *brokersearch.Searcher,
	quotaChecker *service.QuotaChecker,
	log *zap.Logger,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log).
		WithSessionWaiter(hub).
		WithStopGateway(hub.RemoveGateway).
		WithQuotaChecker(quotaChecker)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, withSency(otelInterceptor, authInterceptor)))
}
