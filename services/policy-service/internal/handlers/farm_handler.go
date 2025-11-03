package handlers

import (
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
)

type FarmHandler struct {
	farmService *services.FarmService
	minioClient *minio.Client
}

func NewFarmHandler(farmService *services.FarmService, minioClient *minio.Client) *FarmHandler {
	return &FarmHandler{
		farmService: farmService,
		minioClient: minioClient,
	}
}

func (h *FarmHandler) RegisterRoutes(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	protectedGr.Get("/farms/:id", h.GetFarmByID)
	protectedGr.Post("/farms", h.CreateFarm)
	protectedGr.Put("/farms/:id", h.UpdateFarm)
	protectedGr.Put("/farms/:id", h.DeleteFarm)
}

func (h *FarmHandler) GetFarmByID(c fiber.Ctx) error {
	// get farm id from params
	farmID := c.Params("id")
	userID := c.Get("X-User-ID")

	farm, err := h.farmService.GetFarmByID(farmID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	c.JSON(farm)
	return nil
}

func (h *FarmHandler) CreateFarm(c fiber.Ctx) error {
	var farm models.Farm
	if err := c.JSON(&farm); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Get user ID from header
	userID := c.Get("X-User-ID")
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID is required")
	}

	// Validate required fields
	if farm.CropType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "crop_type is required")
	}
	if farm.AreaSqm <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "area_sqm must be greater than 0")
	}

	// Validate harvest date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return fiber.NewError(fiber.StatusBadRequest, "planting_date is required when expected_harvest_date is provided")
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return fiber.NewError(fiber.StatusBadRequest, "expected_harvest_date must be greater than or equal to planting_date")
		}
	}

	// Create the farm
	if err := h.farmService.CreateFarm(&farm, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(farm)
}

func (h *FarmHandler) UpdateFarm(c fiber.Ctx) error {
	var farm models.Farm
	if err := c.JSON(&farm); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Get user ID from header
	userID := c.Get("X-User-ID")
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID is required")
	}

	// Validate harvest date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return fiber.NewError(fiber.StatusBadRequest, "planting_date is required when expected_harvest_date is provided")
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return fiber.NewError(fiber.StatusBadRequest, "expected_harvest_date must be greater than or equal to planting_date")
		}
	}

	// Update the farm
	if err := h.farmService.UpdateFarm(&farm, userID); err != nil {
		if strings.Contains(err.Error(), "badrequest") {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(farm)
}

func (h *FarmHandler) DeleteFarm(c fiber.Ctx) error {
	farmID := c.Params("id")
	userID := c.Get("X-User-ID")

	if err := h.farmService.DeleteFarm(farmID, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusNoContent).JSON(nil)
}
