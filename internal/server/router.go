package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sarbonNew/internal/admins"
	"sarbonNew/internal/approles"
	"sarbonNew/internal/appusers"
	"sarbonNew/internal/cargo"
	"sarbonNew/internal/chat"
	"sarbonNew/internal/companies"
	"sarbonNew/internal/companytz"
	"sarbonNew/internal/config"
	"sarbonNew/internal/dispatchercompanies"
	"sarbonNew/internal/dispatcherinvitations"
	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/driverinvitations"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/goadmin"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/handlers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/server/swaggerui"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/trips"
)

func NewRouter(cfg config.Config, deps *infra.Infra, logger *zap.Logger) http.Handler {
	if cfg.AppEnv == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mw.RequestLogger(logger, cfg.AppEnv == "local"))

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

	// Вставка ссылки на кастомный CSS в страницы админки (тема не выводит CustomHeadHtml)
	r.Use(goadmin.InjectCSSMiddleware())
	// Обрезка пробелов в query-параметрах для /admin — иначе UUID с пробелом даёт pq: invalid input syntax for type uuid
	r.Use(goadmin.TrimAdminQueryMiddleware())

	// GoAdmin panel at /admin (login: admin / admin)
	if cfg.DatabaseURL != "" {
		if err := goadmin.Mount(r, cfg.DatabaseURL); err != nil {
			logger.Error("goadmin mount failed", zap.Error(err))
		}
	}

	// API v1
	v1 := r.Group("/v1")
	v1.Use(mw.RequireBaseHeaders(cfg))

	driversRepo := drivers.NewRepo(deps.PG)
	dispatchersRepo := dispatchers.NewRepo(deps.PG)
	adminsRepo := admins.NewRepo(deps.PG)
	companiesRepo := companies.NewRepo(deps.PG)
	appusersRepo := appusers.NewRepo(deps.PG)
	cargoRepo := cargo.NewRepo(deps.PG)
	tripsRepo := trips.NewRepo(deps.PG)
	dcrRepo := dispatchercompanies.NewRepo(deps.PG)
	dispInvRepo := dispatcherinvitations.NewRepo(deps.PG)
	driverInvRepo := driverinvitations.NewRepo(deps.PG)
	jwtm := security.NewJWTManager(cfg.JWTSigningKey, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	otpVerifyWindow := time.Duration(cfg.OTPVerifyWindowSeconds) * time.Second
	otpStore := store.NewOTPStore(
		deps.Redis,
		cfg.JWTSigningKey,
		cfg.OTPTTL,
		cfg.OTPResendCooldown,
		cfg.OTPMaxAttempts,
		int64(cfg.OTPSendLimitPerPhonePerHour),
		int64(cfg.OTPSendLimitPerIPPerHour),
		cfg.OTPSendWindow,
		int64(cfg.OTPVerifyAttemptsPerPhone),
		otpVerifyWindow,
	)
	companyUserOTPStore := store.NewOTPStoreWithPrefix(
		deps.Redis,
		cfg.JWTSigningKey,
		cfg.OTPTTL,
		cfg.OTPResendCooldown,
		cfg.OTPMaxAttempts,
		int64(cfg.OTPSendLimitPerPhonePerHour),
		int64(cfg.OTPSendLimitPerIPPerHour),
		cfg.OTPSendWindow,
		int64(cfg.OTPVerifyAttemptsPerPhone),
		otpVerifyWindow,
		"company_",
	)
	sessionStore := store.NewSessionStore(deps.Redis, 15*time.Minute)
	refreshStore := store.NewRefreshStore(deps.Redis, cfg.JWTRefreshTTL, cfg.JWTAccessTTL)
	tgClient := telegram.NewGatewayClient(cfg.TelegramGatewayBaseURL, cfg.TelegramGatewayToken, cfg.TelegramGatewaySenderID, cfg.TelegramGatewayBypass)
	phoneChangeStore := store.NewPhoneChangeStore(deps.Redis, cfg.JWTSigningKey, cfg.OTPTTL, cfg.OTPMaxAttempts)

	dispRegSessions := store.NewDispatcherSessionStore(deps.Redis, "disp_regsession", 15*time.Minute)
	companyUserRegSessions := store.NewDispatcherSessionStore(deps.Redis, "company_regsession", 15*time.Minute)
	dispResetActions := store.NewDispatcherOTPActionStore(deps.Redis, cfg.JWTSigningKey, "disp_reset", cfg.OTPTTL, cfg.OTPMaxAttempts)
	dispPhoneActions := store.NewDispatcherOTPActionStore(deps.Redis, cfg.JWTSigningKey, "disp_phone", cfg.OTPTTL, cfg.OTPMaxAttempts)

	authH := handlers.NewAuthHandler(logger, driversRepo, otpStore, sessionStore, refreshStore, jwtm, tgClient, cfg.OTPTTL, cfg.OTPLength)
	regH := handlers.NewRegistrationHandler(logger, driversRepo, sessionStore, jwtm, refreshStore)
	kycH := handlers.NewKYCHandler(logger, driversRepo)
	profileH := handlers.NewProfileHandler(logger, driversRepo, phoneChangeStore, tgClient, cfg.OTPTTL, cfg.OTPLength)

	dispAuthH := handlers.NewDispatcherAuthHandler(logger, dispatchersRepo, otpStore, dispRegSessions, dispResetActions, jwtm, refreshStore, tgClient, cfg.OTPTTL, cfg.OTPLength)
	dispRegH := handlers.NewDispatcherRegistrationHandler(logger, dispatchersRepo, dispRegSessions, jwtm, refreshStore)
	dispProfileH := handlers.NewDispatcherProfileHandler(logger, dispatchersRepo, dispPhoneActions, tgClient, cfg.OTPTTL, cfg.OTPLength)
	adminAuthH := handlers.NewAdminAuthHandler(logger, adminsRepo, jwtm, refreshStore)
	adminCompaniesH := handlers.NewAdminCompaniesHandler(logger, companiesRepo, appusersRepo)
	cargoH := handlers.NewCargoHandler(logger, cargoRepo, tripsRepo, jwtm, cfg)
	dispCompaniesH := handlers.NewDispatcherCompaniesHandler(logger, companiesRepo, dcrRepo, jwtm)
	dispInvH := handlers.NewDispatcherInvitationsHandler(logger, dispInvRepo, dcrRepo, dispatchersRepo)
	driverInvH := handlers.NewDriverInvitationsHandler(logger, driverInvRepo, dcrRepo, driversRepo)
	tripsH := handlers.NewTripsHandler(logger, tripsRepo)

	chatRepo := chat.NewRepo(deps.PG)
	chatPresence := chat.NewPresenceStore(deps.Redis)
	chatHub := chat.NewHub(chatPresence, logger)
	chatH := handlers.NewChatHandler(logger, chatRepo, chatPresence, chatHub)

	approlesRepo := approles.NewRepo(deps.PG)
	ucrRepo := companytz.NewRepoUCR(deps.PG)
	invitationsRepo := companytz.NewRepoInvitations(deps.PG)
	auditRepo := companytz.NewRepoAudit(deps.PG)
	companyUserAuthH := handlers.NewCompanyUserAuthHandler(logger, appusersRepo, companyUserOTPStore, companyUserRegSessions, jwtm, refreshStore, tgClient, cfg.OTPTTL, cfg.OTPLength)
	companyUserRegH := handlers.NewCompanyUserRegistrationHandler(logger, appusersRepo, companyUserRegSessions, jwtm, refreshStore)
	companyTZH := handlers.NewCompanyTZHandler(logger, appusersRepo, companiesRepo, approlesRepo, ucrRepo, invitationsRepo, auditRepo, jwtm)

	v1.POST("/company-users/auth/phone", companyUserAuthH.SendOTP)
	v1.POST("/company-users/auth/otp/verify", companyUserAuthH.VerifyOTP)
	v1.POST("/company-users/auth/refresh", authH.Refresh)   // company user: обновить пару токенов по refresh_token
	v1.POST("/company-users/registration/complete", companyUserRegH.Complete)

	// Driver: только API водителя (auth, registration, profile, trips, invitations)
	v1.POST("/driver/auth/phone", authH.SendOTP)
	v1.POST("/driver/auth/otp/verify", authH.VerifyOTP)
	v1.POST("/driver/auth/refresh", authH.Refresh)
	v1.POST("/driver/auth/logout", authH.Logout)
	v1.POST("/driver/registration/start", regH.Start)
	v1.GET("/driver/transport-options", handlers.GetTransportOptions)

	// Reference: справочники (общие для водителя, диспетчера и др.)
	v1.GET("/reference/drivers", handlers.GetReferenceDrivers)
	v1.GET("/reference/cargo", handlers.GetReferenceCargo)
	v1.GET("/reference/company", handlers.GetReferenceCompany(approlesRepo))
	v1.GET("/reference/admin", handlers.GetReferenceAdmin)
	v1.GET("/reference/dispatchers", handlers.GetReferenceDispatchers)
	v1.GET("/reference/cities", handlers.GetReferenceCities())
	v1.GET("/reference/countries", handlers.GetReferenceCountries())

	// API /api/cargo (same base headers as v1)
	api := r.Group("/api")
	api.Use(mw.RequireBaseHeaders(cfg))
	api.POST("/cargo", cargoH.Create)
	api.GET("/cargo", cargoH.List)
	api.GET("/cargo/:id", cargoH.GetByID)
	api.PUT("/cargo/:id", cargoH.Update)
	api.DELETE("/cargo/:id", cargoH.Delete)
	api.PATCH("/cargo/:id/status", cargoH.PatchStatus)
	api.POST("/cargo/:id/offers", cargoH.CreateOffer)
	api.GET("/cargo/:id/offers", cargoH.ListOffers)
	api.POST("/offers/:id/accept", cargoH.AcceptOffer)
	api.GET("/trips", tripsH.List)
	api.GET("/trips/:id", tripsH.Get)

	v1.POST("/dispatchers/auth/phone", dispAuthH.SendOTP)
	v1.POST("/dispatchers/auth/otp/verify", dispAuthH.VerifyOTP)
	v1.POST("/dispatchers/auth/login/password", dispAuthH.LoginPassword)
	v1.POST("/dispatchers/auth/refresh", authH.Refresh)   // диспетчер: обновить пару токенов по refresh_token
	v1.POST("/dispatchers/auth/reset-password/request", dispAuthH.ResetPasswordRequest)
	v1.POST("/dispatchers/auth/reset-password/confirm", dispAuthH.ResetPasswordConfirm)
	v1.POST("/dispatchers/auth/logout", dispAuthH.Logout)
	v1.POST("/dispatchers/registration/complete", dispRegH.Complete)

	// Admin auth (login by password, refresh) — только base headers; без admin token
	v1.POST("/admin/auth/login/password", adminAuthH.LoginPassword)
	v1.POST("/admin/auth/refresh", authH.Refresh)   // админ: обновить пару токенов по refresh_token

	// Все маршруты под adminAuthed проверяют: base headers (X-Client-Token, X-Device-Type, X-Language) + X-User-Token с role=admin
	adminAuthed := v1.Group("/admin")
	adminAuthed.Use(mw.RequireAdmin(jwtm, refreshStore))
	adminAuthed.POST("/companies", adminCompaniesH.Create)
	adminAuthed.PATCH("/companies/:id/owner", adminCompaniesH.SetOwner)
	adminAuthed.GET("/company-users/owners/search", adminCompaniesH.SearchOwners)

	driverAuthed := v1.Group("/driver")
	driverAuthed.Use(mw.RequireDriver(jwtm, refreshStore))
	driverAuthed.Use(mw.UpdateDriverLastOnline(driversRepo))
	driverAuthed.GET("/profile", profileH.Get)
	driverAuthed.PATCH("/profile/driver", profileH.PatchDriver)
	driverAuthed.PUT("/profile/heartbeat", profileH.Heartbeat)
	driverAuthed.POST("/profile/photo", profileH.UploadPhoto)
	driverAuthed.GET("/profile/photo", profileH.GetPhoto)
	driverAuthed.DELETE("/profile/photo", profileH.DeletePhoto)
	driverAuthed.POST("/profile/phone-change/request", profileH.PhoneChangeRequest)
	driverAuthed.POST("/profile/phone-change/verify", profileH.PhoneChangeVerify)
	driverAuthed.PATCH("/profile/power", profileH.PatchPower)
	driverAuthed.PATCH("/profile/trailer", profileH.PatchTrailer)
	driverAuthed.DELETE("/profile", profileH.Delete)
	driverAuthed.PATCH("/registration/geo-push", regH.GeoPush)
	driverAuthed.PATCH("/registration/transport-type", regH.TransportType)
	driverAuthed.PATCH("/kyc", kycH.Submit)
	driverAuthed.GET("/trips", tripsH.ListMy)
	driverAuthed.POST("/trips/:id/confirm", tripsH.DriverConfirm)
	driverAuthed.POST("/trips/:id/reject", tripsH.DriverReject)
	driverAuthed.PATCH("/trips/:id/status", tripsH.PatchStatus)
	driverAuthed.GET("/driver-invitations", driverInvH.ListInvitations)
	driverAuthed.POST("/driver-invitations/accept", driverInvH.Accept)
	driverAuthed.POST("/driver-invitations/decline", driverInvH.Decline)

	// Dispatchers: только API диспетчера
	dispAuthed := v1.Group("/dispatchers")
	dispAuthed.Use(mw.RequireDispatcher(jwtm, refreshStore))
	dispAuthed.Use(mw.UpdateDispatcherLastOnline(dispatchersRepo))
	dispAuthed.GET("/profile", dispProfileH.Get)
	dispAuthed.PATCH("/profile", dispProfileH.Patch)
	dispAuthed.POST("/profile/photo", dispProfileH.UploadPhoto)
	dispAuthed.GET("/profile/photo", dispProfileH.GetPhoto)
	dispAuthed.PUT("/profile/password", dispProfileH.ChangePassword)
	dispAuthed.POST("/profile/phone-change/request", dispProfileH.PhoneChangeRequest)
	dispAuthed.POST("/profile/phone-change/verify", dispProfileH.PhoneChangeVerify)
	dispAuthed.DELETE("/profile", dispProfileH.Delete)
	// Freelance: no create company; list/switch only when invited. Cargo/offers/trips via /api and below.
	dispAuthed.GET("/companies", dispCompaniesH.ListMyCompanies)
	dispAuthed.POST("/auth/switch-company", dispCompaniesH.SwitchCompany)
	dispAuthed.POST("/companies/:companyId/invitations", dispInvH.CreateInvitation)
	dispAuthed.POST("/invitations/accept", dispInvH.Accept)
	dispAuthed.POST("/invitations/decline", dispInvH.Decline)
	dispAuthed.GET("/driver-invitations", driverInvH.ListSent)
	dispAuthed.POST("/driver-invitations", driverInvH.CreateForFreelance)
	dispAuthed.DELETE("/driver-invitations/:token", driverInvH.CancelInvitation)
	dispAuthed.POST("/companies/:companyId/driver-invitations", driverInvH.Create)
	dispAuthed.GET("/drivers/find", driverInvH.FindDrivers)
	dispAuthed.GET("/drivers", driverInvH.ListMyDrivers)
	dispAuthed.DELETE("/drivers/:driverId", driverInvH.UnlinkDriver)
	dispAuthed.PUT("/drivers/:driverId/power", driverInvH.SetDriverPower)
	dispAuthed.PUT("/drivers/:driverId/trailer", driverInvH.SetDriverTrailer)
	dispAuthed.PATCH("/trips/:id/assign-driver", tripsH.AssignDriver)

	// Company users (company_users): OTP auth, companies, invitations
	appUserAuthed := v1.Group("")
	appUserAuthed.Use(mw.RequireAppUser(jwtm, refreshStore))
	appUserAuthed.GET("/auth/companies", companyTZH.ListMyCompanies)
	appUserAuthed.POST("/auth/switch-company", companyTZH.SwitchCompany)
	appUserAuthed.POST("/companies", companyTZH.CreateCompany)
	appUserAuthed.POST("/companies/:companyId/invitations", companyTZH.CreateInvitation)
	appUserAuthed.POST("/invitations/accept", companyTZH.AcceptInvitation)
	appUserAuthed.GET("/companies/:companyId/users", companyTZH.ListCompanyUsers)
	appUserAuthed.PUT("/companies/:companyId/users/:userId/role", companyTZH.UpdateUserRole)
	appUserAuthed.DELETE("/companies/:companyId/users/:userId", companyTZH.RemoveUser)

	// Chat (driver, dispatcher, admin): JWT or X-User-ID for Swagger testing; WS supports ?user_id= or ?token=
	chatGroup := v1.Group("/chat")
	chatGroup.Use(mw.RequireChatUser(jwtm, refreshStore))
	chatGroup.GET("/conversations", chatH.ListConversations)
	chatGroup.POST("/conversations", chatH.GetOrCreateConversation)
	chatGroup.GET("/conversations/:id/messages", chatH.ListMessages)
	chatGroup.POST("/conversations/:id/messages", chatH.SendMessage)
	chatGroup.PATCH("/messages/:id", chatH.EditMessage)
	chatGroup.DELETE("/messages/:id", chatH.DeleteMessage)
	chatGroup.GET("/presence/:user_id", chatH.GetPresence)
	chatGroup.GET("/ws", chatH.ServeWS)

	return r
}
