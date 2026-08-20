package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/unknowncode44/appointments/internal/api"
	"github.com/unknowncode44/appointments/internal/api/handlers"
	"github.com/unknowncode44/appointments/internal/api/middleware"
	"github.com/unknowncode44/appointments/internal/repositories"
	"github.com/unknowncode44/appointments/internal/services"
)

func SetupRoutes(server *api.Server) error {

	// ── Health endpoints (no auth) ──────────────────────────────────────────
	server.App.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	server.App.Get("/readyz", func(c *fiber.Ctx) error {
		if err := server.Store.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "db unavailable"})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// ── Public auth endpoints ───────────────────────────────────────────────
	userHandler := handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config)

	server.App.Post("/register", userHandler.RegisterUser)
	server.App.Post("/login", userHandler.LoginUser)
	server.App.Post("/tokens/renew_access", handlers.NewTokenHandler(server.Store, server.TokenMaker, server.Config).RenewAccessToken)

	// ── Domain handler instances (one each) ─────────────────────────────────
	adminRepo := repositories.NewAdminRepository(server.Store)
	schedulingRepo := repositories.NewSchedulingRepository(server.Store)
	workflowRepo := repositories.NewWorkflowRepository(server.Store)

	adminHandler := handlers.NewAdminMVPHandler(services.NewAdminService(adminRepo))
	schedulingHandler := handlers.NewSchedulingMVPHandler(services.NewSchedulingService(schedulingRepo))
	appointmentHandler := handlers.NewAppointmentMVPHandler(services.NewAppointmentService(workflowRepo))
	conversationHandler := handlers.NewConversationMVPHandler(services.NewConversationService(workflowRepo))
	evolutionHandler := handlers.NewN8NWhatsAppHandler(services.NewEvolutionService(server.Config))

	// DISABLED: 	// ── WhatsApp proxy (tenantUser only) ───────────────────────────────────
	// DISABLED: 	whatsappHandler := handlers.NewWhatsappHandler(server.Store, server.Config)

	// ── Webhook (public — Evolution calls without JWT) ──────────────────────
	server.App.Post("/api/v1/webhooks/evolution", conversationHandler.EvolutionWebhook)

	// ═══════════════════════════════════════════════════════════════════════
	// N8N INTEGRATION ROUTES (API Key auth only, NO JWT required)
	// ═══════════════════════════════════════════════════════════════════════
	// Routes under /api/v1/n8n/* are exclusively for N8N integration.
	// Authentication: X-API-Key header (validated by APIKeyMiddleware)
	// Tenant isolation: X-Tenant-ID header or tenant_id in body/query (validated by ValidateTenantFromRequest)
	// Rate limited separately at 500 req/min (configured in server.go)

	n8n := server.App.Group("/api/v1/n8n",
		middleware.APIKeyMiddleware(server.Config),
		middleware.ValidateTenantFromRequest(server.Store))

	// GET - Query endpoints (tenant_id from X-Tenant-ID header or query param)
	n8n.Get("/services", adminHandler.ListServices)
	n8n.Get("/providers", adminHandler.ListProviders)
	n8n.Get("/customers", adminHandler.ListCustomers)
	n8n.Get("/availability", schedulingHandler.Availability)
	n8n.Get("/appointments", appointmentHandler.List)
	n8n.Get("/appointments/:id", appointmentHandler.Get)

	// POST - Create endpoints (tenant_id from X-Tenant-ID header or body)
	n8n.Post("/customers", adminHandler.CreateCustomer)
	n8n.Post("/appointments", appointmentHandler.Create)
	n8n.Post("/inbound-messages", conversationHandler.InboundMessage)
	n8n.Post("/whatsapp/send", evolutionHandler.SendText)

	// PUT - Update endpoints
	n8n.Put("/conversation-state/:customer_id", conversationHandler.UpdateConversationState)

	// PATCH/DELETE - Modify endpoints
	n8n.Patch("/appointments/:id", appointmentHandler.Update)
	n8n.Delete("/appointments/:id", appointmentHandler.Delete)

	// ═══════════════════════════════════════════════════════════════════════
	// JWT AUTHENTICATED ROUTES (original routes, unchanged)
	// ═══════════════════════════════════════════════════════════════════════

	// ── Authenticated group ─────────────────────────────────────────────────
	auth := server.App.Group("/", middleware.AuthMiddleware(server.TokenMaker))

	// Own-user routes (any authenticated role)
	auth.Get("/user/info", userHandler.GetUserProfile)
	auth.Put("/user/update", userHandler.UpdateUserProfile)
	auth.Post("/user/password_change", userHandler.ChangePassword)

	v1 := auth.Group("/api/v1")

	// ── Role shortcuts ───────────────────────────────────────────────────────
	adminOnly := middleware.RequireRole("adminUser")
	adminOrTenant := middleware.RequireRole("adminUser", "tenantUser")

	// ── Admin-only: tenant management ───────────────────────────────────────
	v1.Get("/tenants", adminOnly, adminHandler.ListTenants)
	v1.Post("/tenants", adminOnly, adminHandler.CreateTenant)
	v1.Get("/tenants/:id", adminOnly, adminHandler.GetTenant)
	v1.Put("/tenants/:id", adminOnly, adminHandler.UpdateTenant)
	v1.Put("/tenants/:id/greeting", adminOrTenant, adminHandler.UpdateTenantGreeting)
	v1.Delete("/tenants/:id", adminOnly, adminHandler.DeactivateTenant)

	// ── User management ──────────────────────────────────────────────────────
	// POST /users: adminUser has full control; tenantUser can create role=user in their tenant.
	// All other user routes remain admin-only.
	v1.Get("/users", adminOnly, userHandler.ListUsers)
	v1.Post("/users", adminOrTenant, userHandler.CreateUserAdmin)
	v1.Get("/users/:id", adminOnly, userHandler.GetUserByID)
	v1.Put("/users/:id", adminOnly, userHandler.UpdateUser)
	v1.Delete("/users/:id", adminOnly, userHandler.DeleteUser)
	v1.Post("/users/:id/tenant", adminOnly, userHandler.LinkUserToTenant)
	v1.Get("/users/:id/providers", adminOnly, userHandler.GetUserProviders)

	// Admin + tenantUser: user-provider links
	v1.Post("/users/:id/provider", adminOrTenant, userHandler.LinkUserToProvider)
	v1.Delete("/users/:id/provider", adminOrTenant, userHandler.RemoveUserFromProvider)

	// ── Admin + tenantUser (tenant-isolated) ────────────────────────────────
	// RequireTenant ensures tenantUser can only access their own tenant's data.
	v1.Get("/providers", adminOrTenant, middleware.RequireTenant("tenant_id"), adminHandler.ListProviders)
	v1.Post("/providers", adminOrTenant, adminHandler.CreateProvider)
	v1.Get("/providers/:id", adminOrTenant, adminHandler.GetProvider)
	v1.Put("/providers/:id", adminOrTenant, adminHandler.UpdateProvider)
	v1.Delete("/providers/:id", adminOrTenant, adminHandler.DeactivateProvider)

	v1.Post("/providers/:id/availability", adminOrTenant, schedulingHandler.CreateAvailability)
	v1.Get("/providers/:id/availability", adminOrTenant, schedulingHandler.ListAvailability)
	v1.Post("/providers/:id/exceptions", adminOrTenant, schedulingHandler.CreateException)
	v1.Get("/providers/:id/exceptions", adminOrTenant, schedulingHandler.ListExceptions)

	v1.Get("/services", adminOrTenant, middleware.RequireTenant("tenant_id"), adminHandler.ListServices)
	v1.Post("/services", adminOrTenant, adminHandler.CreateService)
	v1.Get("/services/:id", adminOrTenant, adminHandler.GetService)
	v1.Put("/services/:id", adminOrTenant, adminHandler.UpdateService)
	v1.Delete("/services/:id", adminOrTenant, adminHandler.DeactivateService)

	v1.Get("/tenant-channels", adminOrTenant, middleware.RequireTenant("tenant_id"), adminHandler.ListTenantChannels)
	v1.Post("/tenant-channels", adminOrTenant, adminHandler.CreateTenantChannel)
	v1.Get("/tenant-channels/:id", adminOrTenant, adminHandler.GetTenantChannel)
	v1.Put("/tenant-channels/:id", adminOrTenant, adminHandler.UpdateTenantChannel)
	v1.Delete("/tenant-channels/:id", adminOrTenant, adminHandler.DeactivateTenantChannel)

	// ── All authenticated users ─────────────────────────────────────────────
	allRoles := middleware.RequireRole("adminUser", "tenantUser", "user")

	v1.Get("/customers", allRoles, middleware.RequireTenant("tenant_id"), adminHandler.ListCustomers)
	v1.Post("/customers", allRoles, adminHandler.CreateCustomer)
	v1.Get("/customers/:id", allRoles, adminHandler.GetCustomer)
	v1.Put("/customers/:id", allRoles, adminHandler.UpdateCustomer)

	v1.Post("/slot-generator", allRoles, schedulingHandler.GenerateSlots)
	v1.Get("/availability", allRoles, middleware.RequireTenant("tenant_id"), schedulingHandler.Availability)

	v1.Post("/appointments", allRoles, appointmentHandler.Create)
	v1.Get("/appointments", allRoles, middleware.RequireTenant("tenant_id"), appointmentHandler.List)
	v1.Get("/appointments/:id", allRoles, appointmentHandler.Get)
	v1.Patch("/appointments/:id", allRoles, appointmentHandler.Update)
	v1.Delete("/appointments/:id", allRoles, appointmentHandler.Delete)

	v1.Get("/conversations", adminOrTenant, middleware.RequireTenant("tenant_id"), conversationHandler.List)
	v1.Get("/conversations/:id", adminOrTenant, conversationHandler.Get)
	v1.Post("/conversations/message", adminOrTenant, conversationHandler.Message)
	v1.Post("/inbound-messages", adminOrTenant, conversationHandler.InboundMessage)

	// DISABLED: 	// ── WhatsApp proxy (tenantUser only) ────────────────────────────────────
	// DISABLED: 	tenantOnly := middleware.RequireRole("tenantUser")
	// DISABLED: 	wa := v1.Group("/whatsapp", tenantOnly)
	// DISABLED: 	wa.Post("/instance", whatsappHandler.CreateInstance)
	// DISABLED: 	wa.Get("/instance/status", whatsappHandler.GetStatus)
	// DISABLED: 	wa.Get("/instance/qr", whatsappHandler.GetQR)
	// DISABLED: 	wa.Delete("/instance/logout", whatsappHandler.Logout)
	// DISABLED: 	wa.Delete("/instance", whatsappHandler.DeleteInstance)

	return nil
}
