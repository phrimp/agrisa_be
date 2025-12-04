package handlers

import (
	utils "agrisa_utils"
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"

	"github.com/gofiber/fiber/v3"
)

type DashboardHandler struct {
	DashboardService *services.DashboardService
}

func NewDashboardHandler(dashboardService *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		DashboardService: dashboardService,
	}
}

func (h *DashboardHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	dashboardGr := protectedGr.Group("/dashboard")

	// Partner routes

	// Admin routes
	dashboardGr.Post("/admin/revenue-overview", h.GetAdminRevenueOverview)
}

func (h *DashboardHandler) GetAdminRevenueOverview(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	var req *models.MonthlyRevenueOptions
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("BAD_REQUEST", "Invalid request body"))
	}

	overview, err := h.DashboardService.GetAdminRevenueOverview(*req)
	if err != nil {
		slog.Error("failed to get admin revenue overview", "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "Failed to get revenue overview"))
	}
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(overview))
}
