package handlers

import (
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"

	utils "agrisa_utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type DataTierHandler struct {
	dataTierService *services.DataTierService
}

func NewDataTierHandler(dataTierService *services.DataTierService) *DataTierHandler {
	return &DataTierHandler{
		dataTierService: dataTierService,
	}
}

func (dth *DataTierHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Data Tier Category routes
	categoryGroup := protectedGr.Group("/data-tier-categories")
	categoryGroup.Post("/", dth.CreateDataTierCategory)
	categoryGroup.Get("/", dth.GetAllDataTierCategories)
	categoryGroup.Get("/:id", dth.GetDataTierCategoryByID)
	categoryGroup.Put("/:id", dth.UpdateDataTierCategory)
	categoryGroup.Delete("/:id", dth.DeleteDataTierCategory)

	// Data Tier routes
	tierGroup := protectedGr.Group("/data-tiers")
	tierGroup.Post("/", dth.CreateDataTier)
	tierGroup.Get("/", dth.GetAllDataTiers)
	tierGroup.Get("/:id", dth.GetDataTierByID)
	tierGroup.Put("/:id", dth.UpdateDataTier)
	tierGroup.Delete("/:id", dth.DeleteDataTier)
	tierGroup.Get("/category/:categoryId", dth.GetDataTiersByCategoryID)
	tierGroup.Get("/:id/with-category", dth.GetDataTierWithCategory)
	tierGroup.Get("/:id/total-multiplier", dth.CalculateTotalMultiplier)
}

func (dth *DataTierHandler) CreateDataTierCategory(c fiber.Ctx) error {
	var req models.CreateDataTierCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	category, err := dth.dataTierService.CreateDataTierCategory(req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CREATION_FAILED", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(category))
}

func (dth *DataTierHandler) GetAllDataTierCategories(c fiber.Ctx) error {
	categories, err := dth.dataTierService.GetAllDataTierCategories()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(categories))
}

func (dth *DataTierHandler) GetDataTierCategoryByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	category, err := dth.dataTierService.GetDataTierCategoryByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(category))
}

func (dth *DataTierHandler) UpdateDataTierCategory(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	var req models.UpdateDataTierCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	category, err := dth.dataTierService.UpdateDataTierCategory(id, req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(category))
}

func (dth *DataTierHandler) DeleteDataTierCategory(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	err = dth.dataTierService.DeleteDataTierCategory(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("DELETE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]string{
		"message": "Data tier category deleted successfully",
	}))
}

func (dth *DataTierHandler) CreateDataTier(c fiber.Ctx) error {
	var req models.CreateDataTierRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	tier, err := dth.dataTierService.CreateDataTier(req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CREATION_FAILED", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(tier))
}

func (dth *DataTierHandler) GetAllDataTiers(c fiber.Ctx) error {
	tiers, err := dth.dataTierService.GetAllDataTiers()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(tiers))
}

func (dth *DataTierHandler) GetDataTierByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	tier, err := dth.dataTierService.GetDataTierByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(tier))
}

func (dth *DataTierHandler) UpdateDataTier(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	var req models.UpdateDataTierRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	tier, err := dth.dataTierService.UpdateDataTier(id, req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(tier))
}

func (dth *DataTierHandler) DeleteDataTier(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	err = dth.dataTierService.DeleteDataTier(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("DELETE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]string{
		"message": "Data tier deleted successfully",
	}))
}

func (dth *DataTierHandler) GetDataTiersByCategoryID(c fiber.Ctx) error {
	categoryIdParam := c.Params("categoryId")
	categoryId, err := uuid.Parse(categoryIdParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	tiers, err := dth.dataTierService.GetDataTiersByCategoryID(categoryId)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(tiers))
}

func (dth *DataTierHandler) GetDataTierWithCategory(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	tier, category, err := dth.dataTierService.GetDataTierWithCategory(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
	}

	response := map[string]interface{}{
		"tier":     tier,
		"category": category,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (dth *DataTierHandler) CalculateTotalMultiplier(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	multiplier, err := dth.dataTierService.CalculateTotalMultiplier(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CALCULATION_FAILED", err.Error()))
	}

	response := map[string]interface{}{
		"tier_id":          id,
		"total_multiplier": multiplier,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}
