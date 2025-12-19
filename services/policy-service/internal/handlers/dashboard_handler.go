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
	dashboardGr.Post("/partner/overview", h.GetPartnerDashboardOverview)

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

func (h *DashboardHandler) GetPartnerDashboardOverview(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	var req *models.PartnerDashboardRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("BAD_REQUEST", "Invalid request body"))
	}

	// Validate required fields
	if req.PartnerID == "" {
		slog.Error("partner_id is required", "user_id", userID)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("BAD_REQUEST", "partner_id is required"))
	}

	if req.StartDate == 0 || req.EndDate == 0 {
		slog.Error("start_date and end_date are required", "user_id", userID)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("BAD_REQUEST", "start_date and end_date are required"))
	}

	if req.StartDate >= req.EndDate {
		slog.Error("start_date must be less than end_date", "user_id", userID, "start_date", req.StartDate, "end_date", req.EndDate)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("BAD_REQUEST", "start_date must be less than end_date"))
	}

	overview, err := h.DashboardService.GetPartnerDashboardOverview(*req)
	if err != nil {
		slog.Error("failed to get partner dashboard overview", "user_id", userID, "partner_id", req.PartnerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "Failed to get dashboard overview"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(overview))
}
