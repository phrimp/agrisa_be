package handlers

import (
	utils "agrisa_utils"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type FarmHandler struct {
	farmService *services.FarmService
	minioClient *minio.MinioClient
}

func NewFarmHandler(farmService *services.FarmService, minioClient *minio.MinioClient) *FarmHandler {
	return &FarmHandler{
		farmService: farmService,
		minioClient: minioClient,
	}
}

func (h *FarmHandler) RegisterRoutes(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	protectedGr.Get("/farms/me", h.GetFarmByOwner)
	protectedGr.Get("/farms/:id", h.GetFarmByID)
	protectedGr.Post("/farms", h.CreateFarm)
	protectedGr.Put("/farms/:id", h.UpdateFarm)
	protectedGr.Post("/farms/:id", h.DeleteFarm)
	protectedGr.Get("/farms", h.GetAllFarms)
}

// func (h *FarmHandler) GetFarmByOwner(c fiber.Ctx) error {
// 	// get farm id from params
// 	userID := c.Get("X-User-ID")

// 	farms, err := h.farmService.GetFarmByOwnerID(c.Context(), userID)
// 	if err != nil {
// 		if strings.Contains(err.Error(), "unauthorized") {
// 			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
// 		}
// 		if strings.Contains(err.Error(), "not found") {
// 			return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
// 		}
// 		if strings.Contains(err.Error(), "invalid") {
// 			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", err.Error()))
// 		}
// 		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
// 	}
// 	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(farms))
// }

func (h *FarmHandler) GetFarmByOwner(c fiber.Ctx) error {
	// get farm id from params
	userID := c.Get("X-User-ID")

	cropType := c.Query("crop_type")

	farms, err := h.farmService.GetFarmByOwnerID(c.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
		}
		if strings.Contains(err.Error(), "invalid") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	filter := []models.Farm{}
	for _, farm := range farms {
		if cropType == "" || strings.EqualFold(farm.CropType, cropType) {
			filter = append(filter, farm)
		}
	}

	farms = filter

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(farms))
}

func (h *FarmHandler) GetFarmByID(c fiber.Ctx) error {
	// get farm id from params
	farmID := c.Params("id")

	farm, err := h.farmService.GetByFarmID(c.Context(), farmID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
		}
		if strings.Contains(err.Error(), "invalid") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(farm))
}

func (h *FarmHandler) CreateFarm(c fiber.Ctx) error {
	var farm models.Farm
	if err := c.Bind().JSON(&farm); err != nil {
		slog.Error("error parsing request", "error", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Get user ID from header
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	token, err := extractBearerToken(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
	}

	err = h.farmService.CreateFarmValidate(&farm, token)
	if err != nil {
		if strings.Contains(err.Error(), "bad_request") {
			log.Printf("Error logginggg: %s", err.Error())
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		if strings.Contains(err.Error(), "forbidden") {
			return c.Status(http.StatusForbidden).JSON(utils.CreateErrorResponse("FORBIDDEN", err.Error()))
		}
		if strings.Contains(err.Error(), "User has no associated national ID card") {
			return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", "User has no associated national ID card"))
		}
		log.Printf("Error logginggg: %s", err.Error())
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	// Create the farm
	if err := h.farmService.CreateFarm(&farm, userID); err != nil {
		log.Println("Error creating farm:", err)
		if strings.Contains(err.Error(), "badrequest") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", err.Error()))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(farm))
}

func (h *FarmHandler) UpdateFarm(c fiber.Ctx) error {
	var farm models.Farm
	if err := c.Bind().JSON(&farm); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "Invalid request body"))
	}

	// Get user ID from header
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Get farm ID from params
	farmID := c.Params("id")

	// Validate harvest date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "planting_date is required when expected_harvest_date is provided"))
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "expected_harvest_date must be greater than or equal to planting_date"))
		}
	}

	// Update the farm
	if err := h.farmService.UpdateFarm(c.Context(), &farm, userID, farmID); err != nil {
		if strings.Contains(err.Error(), "badrequest") {
			return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", err.Error()))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(farm))
}

func (h *FarmHandler) DeleteFarm(c fiber.Ctx) error {
	farmID := c.Params("id")
	userID := c.Get("X-User-ID")

	if err := h.farmService.DeleteFarm(c.Context(), farmID, userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(nil))
}

func (h *FarmHandler) GetAllFarms(c fiber.Ctx) error {
	farms, err := h.farmService.GetAllFarms(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", err.Error()))
	}
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(farms))
}

func extractBearerToken(c fiber.Ctx) (string, error) {
	authHeader := c.Get("Authorization")

	if authHeader == "" {
		log.Printf("authorization header is missing")
		return "", errors.New("authorization header is missing")
	}

	// Kiểm tra format "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		log.Printf("invalid authorization format")
		return "", errors.New("invalid authorization format")
	}

	token := parts[1]
	if token == "" {
		log.Printf("empty token")
		return "", errors.New("empty token")
	}

	return token, nil
}
