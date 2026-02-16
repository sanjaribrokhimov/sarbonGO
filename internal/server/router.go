package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sarbonNew/internal/config"
	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/handlers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/server/swaggerui"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
)

func NewRouter(cfg config.Config, deps *infra.Infra, logger *zap.Logger) http.Handler {
	if cfg.AppEnv == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mw.RequestLogger(logger))

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

	// Public endpoints that should still validate base headers.
	r.GET("/health", func(c *gin.Context) {
		resp.OK(c, gin.H{"status": "ok"})
	})

	// Swagger UI (OpenAPI served from local file)
	swaggerui.Register(r)

	// API v1
	v1 := r.Group("/v1")
	v1.Use(mw.RequireBaseHeaders(cfg))

	driversRepo := drivers.NewRepo(deps.PG)
	dispatchersRepo := dispatchers.NewRepo(deps.PG)
	jwtm := security.NewJWTManager(cfg.JWTSigningKey, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	otpStore := store.NewOTPStore(deps.Redis, cfg.JWTSigningKey, cfg.OTPTTL, cfg.OTPResendCooldown, cfg.OTPMaxAttempts)
	sessionStore := store.NewSessionStore(deps.Redis, 15*time.Minute)
	refreshStore := store.NewRefreshStore(deps.Redis, cfg.JWTRefreshTTL)
	tgClient := telegram.NewGatewayClient(cfg.TelegramGatewayBaseURL, cfg.TelegramGatewayToken, cfg.TelegramGatewaySenderID)
	phoneChangeStore := store.NewPhoneChangeStore(deps.Redis, cfg.JWTSigningKey, cfg.OTPTTL, cfg.OTPMaxAttempts)

	dispRegSessions := store.NewDispatcherSessionStore(deps.Redis, "disp_regsession", 15*time.Minute)
	dispResetActions := store.NewDispatcherOTPActionStore(deps.Redis, cfg.JWTSigningKey, "disp_reset", cfg.OTPTTL, cfg.OTPMaxAttempts)
	dispPhoneActions := store.NewDispatcherOTPActionStore(deps.Redis, cfg.JWTSigningKey, "disp_phone", cfg.OTPTTL, cfg.OTPMaxAttempts)

	authH := handlers.NewAuthHandler(logger, driversRepo, otpStore, sessionStore, refreshStore, jwtm, tgClient, cfg.OTPTTL, cfg.OTPLength)
	regH := handlers.NewRegistrationHandler(logger, driversRepo, sessionStore, jwtm, refreshStore)
	kycH := handlers.NewKYCHandler(logger, driversRepo)
	profileH := handlers.NewProfileHandler(logger, driversRepo, phoneChangeStore, tgClient, cfg.OTPTTL, cfg.OTPLength)

	dispAuthH := handlers.NewDispatcherAuthHandler(logger, dispatchersRepo, otpStore, dispRegSessions, dispResetActions, jwtm, refreshStore, tgClient, cfg.OTPTTL, cfg.OTPLength)
	dispRegH := handlers.NewDispatcherRegistrationHandler(logger, dispatchersRepo, dispRegSessions, jwtm, refreshStore)
	dispProfileH := handlers.NewDispatcherProfileHandler(logger, dispatchersRepo, dispPhoneActions, tgClient, cfg.OTPTTL, cfg.OTPLength)

	v1.POST("/auth/phone", authH.SendOTP)
	v1.POST("/auth/otp/verify", authH.VerifyOTP)
	v1.POST("/auth/refresh", authH.Refresh)
	v1.POST("/auth/logout", authH.Logout)

	v1.POST("/registration/start", regH.Start)

	v1.POST("/dispatchers/auth/phone", dispAuthH.SendOTP)
	v1.POST("/dispatchers/auth/otp/verify", dispAuthH.VerifyOTP)
	v1.POST("/dispatchers/auth/login/password", dispAuthH.LoginPassword)
	v1.POST("/dispatchers/auth/reset-password/request", dispAuthH.ResetPasswordRequest)
	v1.POST("/dispatchers/auth/reset-password/confirm", dispAuthH.ResetPasswordConfirm)
	v1.POST("/dispatchers/registration/complete", dispRegH.Complete)

	authed := v1.Group("")
	authed.Use(mw.RequireDriver(jwtm))
	authed.GET("/profile", profileH.Get)
	authed.PATCH("/profile/driver", profileH.PatchDriver)
	authed.PUT("/profile/heartbeat", profileH.Heartbeat)
	authed.POST("/profile/phone-change/request", profileH.PhoneChangeRequest)
	authed.POST("/profile/phone-change/verify", profileH.PhoneChangeVerify)
	authed.PATCH("/profile/power", profileH.PatchPower)
	authed.PATCH("/profile/trailer", profileH.PatchTrailer)
	authed.DELETE("/profile", profileH.Delete)
	authed.PATCH("/registration/geo-push", regH.GeoPush)
	authed.PATCH("/registration/transport-type", regH.TransportType)
	authed.PATCH("/kyc", kycH.Submit)

	dispAuthed := v1.Group("/dispatchers")
	dispAuthed.Use(mw.RequireDispatcher(jwtm))
	dispAuthed.GET("/profile", dispProfileH.Get)
	dispAuthed.PATCH("/profile", dispProfileH.Patch)
	dispAuthed.PUT("/profile/password", dispProfileH.ChangePassword)
	dispAuthed.POST("/profile/phone-change/request", dispProfileH.PhoneChangeRequest)
	dispAuthed.POST("/profile/phone-change/verify", dispProfileH.PhoneChangeVerify)
	dispAuthed.DELETE("/profile", dispProfileH.Delete)

	return r
}

