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

type DataSourceHandler struct {
	dataSourceService *services.DataSourceService
}

func NewDataSourceHandler(dataSourceService *services.DataSourceService) *DataSourceHandler {
	return &DataSourceHandler{
		dataSourceService: dataSourceService,
	}
}

func (dsh *DataSourceHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Data Source routes
	dataSourceGroup := protectedGr.Group("/data-sources")
	dataSourceGroup.Post("/", dsh.CreateDataSource)
	dataSourceGroup.Post("/batch", dsh.CreateDataSourcesBatch)
	dataSourceGroup.Get("/", dsh.GetAllDataSources)
	dataSourceGroup.Get("/active", dsh.GetActiveDataSources)
	dataSourceGroup.Get("/search", dsh.GetDataSourcesWithFilters)
	dataSourceGroup.Get("/:id", dsh.GetDataSourceByID)
	dataSourceGroup.Put("/:id", dsh.UpdateDataSource)
	dataSourceGroup.Delete("/:id", dsh.DeleteDataSource)
	dataSourceGroup.Patch("/:id/activate", dsh.ActivateDataSource)
	dataSourceGroup.Patch("/:id/deactivate", dsh.DeactivateDataSource)
	dataSourceGroup.Get("/type/:type", dsh.GetDataSourcesByType)
	dataSourceGroup.Get("/tier/:tierId", dsh.GetDataSourcesByTierID)
	dataSourceGroup.Get("/parameter/:parameterName", dsh.GetDataSourcesByParameterName)

	// Utility routes
	dataSourceGroup.Get("/count/total", dsh.GetDataSourceCount)
	dataSourceGroup.Get("/count/active", dsh.GetActiveDataSourceCount)
	dataSourceGroup.Get("/count/type/:type", dsh.GetDataSourceCountByType)
	dataSourceGroup.Get("/count/tier/:tierId", dsh.GetDataSourceCountByTier)
	dataSourceGroup.Get("/:id/exists", dsh.CheckDataSourceExists)
}

// ============================================================================
// CREATE OPERATIONS
// ============================================================================

func (dsh *DataSourceHandler) CreateDataSource(c fiber.Ctx) error {
	var req models.CreateDataSourceRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Convert request to model
	dataSource := &models.DataSource{
		DataSource:        req.DataSource,
		ParameterName:     req.ParameterName,
		ParameterType:     req.ParameterType,
		Unit:              req.Unit,
		DisplayNameVi:     req.DisplayNameVi,
		DescriptionVi:     req.DescriptionVi,
		MinValue:          req.MinValue,
		MaxValue:          req.MaxValue,
		UpdateFrequency:   req.UpdateFrequency,
		SpatialResolution: req.SpatialResolution,
		AccuracyRating:    req.AccuracyRating,
		BaseCost:          req.BaseCost,
		DataTierID:        req.DataTierID,
		DataProvider:      req.DataProvider,
		APIEndpoint:       req.APIEndpoint,
	}

	err := dsh.dataSourceService.CreateDataSource(dataSource)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CREATION_FAILED", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(dataSource))
}

func (dsh *DataSourceHandler) CreateDataSourcesBatch(c fiber.Ctx) error {
	var req models.CreateDataSourceBatchRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Convert requests to models
	dataSources := make([]models.DataSource, len(req.DataSources))
	for i, dsReq := range req.DataSources {
		dataSources[i] = models.DataSource{
			DataSource:        dsReq.DataSource,
			ParameterName:     dsReq.ParameterName,
			ParameterType:     dsReq.ParameterType,
			Unit:              dsReq.Unit,
			DisplayNameVi:     dsReq.DisplayNameVi,
			DescriptionVi:     dsReq.DescriptionVi,
			MinValue:          dsReq.MinValue,
			MaxValue:          dsReq.MaxValue,
			UpdateFrequency:   dsReq.UpdateFrequency,
			SpatialResolution: dsReq.SpatialResolution,
			AccuracyRating:    dsReq.AccuracyRating,
			BaseCost:          dsReq.BaseCost,
			DataTierID:        dsReq.DataTierID,
			DataProvider:      dsReq.DataProvider,
			APIEndpoint:       dsReq.APIEndpoint,
		}
	}

	err := dsh.dataSourceService.CreateDataSourcesBatch(dataSources)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("BATCH_CREATION_FAILED", err.Error()))
	}

	response := map[string]any{
		"message":      "Data sources created successfully",
		"count":        len(dataSources),
		"data_sources": dataSources,
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(response))
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

func (dsh *DataSourceHandler) GetDataSourceByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	dataSource, err := dsh.dataSourceService.GetDataSourceByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSource))
}

func (dsh *DataSourceHandler) GetAllDataSources(c fiber.Ctx) error {
	dataSources, err := dsh.dataSourceService.GetAllDataSources()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

func (dsh *DataSourceHandler) GetActiveDataSources(c fiber.Ctx) error {
	dataSources, err := dsh.dataSourceService.GetActiveDataSources()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

func (dsh *DataSourceHandler) GetDataSourcesByType(c fiber.Ctx) error {
	typeParam := c.Params("type")
	dataSourceType := models.DataSourceType(typeParam)

	dataSources, err := dsh.dataSourceService.GetDataSourcesByType(dataSourceType)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

func (dsh *DataSourceHandler) GetDataSourcesByTierID(c fiber.Ctx) error {
	tierIdParam := c.Params("tierId")
	tierId, err := uuid.Parse(tierIdParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	dataSources, err := dsh.dataSourceService.GetDataSourcesByTierID(tierId)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

func (dsh *DataSourceHandler) GetDataSourcesByParameterName(c fiber.Ctx) error {
	parameterName := c.Params("parameterName")
	if parameterName == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Parameter name cannot be empty"))
	}

	dataSources, err := dsh.dataSourceService.GetDataSourcesByParameterName(parameterName)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

func (dsh *DataSourceHandler) GetDataSourcesWithFilters(c fiber.Ctx) error {
	var req models.DataSourceFiltersRequest
	if err := c.Bind().Query(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid query parameters"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Convert request to service filters
	filters := services.DataSourceFilters{
		TierID:         req.TierID,
		DataSourceType: req.DataSourceType,
		ParameterName:  req.ParameterName,
		ActiveOnly:     req.ActiveOnly,
		MinCost:        req.MinCost,
		MaxCost:        req.MaxCost,
		MinAccuracy:    req.MinAccuracy,
	}

	dataSources, err := dsh.dataSourceService.GetDataSourcesWithFilters(filters)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(dataSources))
}

// ============================================================================
// UPDATE OPERATIONS
// ============================================================================

func (dsh *DataSourceHandler) UpdateDataSource(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	var req models.UpdateDataSourceRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Get existing data source
	existingDataSource, err := dsh.dataSourceService.GetDataSourceByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.CreateErrorResponse("NOT_FOUND", err.Error()))
	}

	// Update fields if provided
	if req.DataSource != nil {
		existingDataSource.DataSource = *req.DataSource
	}
	if req.ParameterName != nil {
		existingDataSource.ParameterName = *req.ParameterName
	}
	if req.ParameterType != nil {
		existingDataSource.ParameterType = *req.ParameterType
	}
	if req.Unit != nil {
		existingDataSource.Unit = req.Unit
	}
	if req.DisplayNameVi != nil {
		existingDataSource.DisplayNameVi = req.DisplayNameVi
	}
	if req.DescriptionVi != nil {
		existingDataSource.DescriptionVi = req.DescriptionVi
	}
	if req.MinValue != nil {
		existingDataSource.MinValue = req.MinValue
	}
	if req.MaxValue != nil {
		existingDataSource.MaxValue = req.MaxValue
	}
	if req.UpdateFrequency != nil {
		existingDataSource.UpdateFrequency = req.UpdateFrequency
	}
	if req.SpatialResolution != nil {
		existingDataSource.SpatialResolution = req.SpatialResolution
	}
	if req.AccuracyRating != nil {
		existingDataSource.AccuracyRating = req.AccuracyRating
	}
	if req.BaseCost != nil {
		existingDataSource.BaseCost = *req.BaseCost
	}
	if req.DataTierID != nil {
		existingDataSource.DataTierID = *req.DataTierID
	}
	if req.DataProvider != nil {
		existingDataSource.DataProvider = req.DataProvider
	}
	if req.APIEndpoint != nil {
		existingDataSource.APIEndpoint = req.APIEndpoint
	}
	if req.IsActive != nil {
		existingDataSource.IsActive = *req.IsActive
	}

	err = dsh.dataSourceService.UpdateDataSource(id, existingDataSource)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(existingDataSource))
}

func (dsh *DataSourceHandler) ActivateDataSource(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	err = dsh.dataSourceService.ActivateDataSource(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("ACTIVATION_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"message":        "Data source activated successfully",
		"data_source_id": id,
		"is_active":      true,
	}))
}

func (dsh *DataSourceHandler) DeactivateDataSource(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	err = dsh.dataSourceService.DeactivateDataSource(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("DEACTIVATION_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"message":        "Data source deactivated successfully",
		"data_source_id": id,
		"is_active":      false,
	}))
}

// ============================================================================
// DELETE OPERATIONS
// ============================================================================

func (dsh *DataSourceHandler) DeleteDataSource(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	err = dsh.dataSourceService.DeleteDataSource(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("DELETE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]string{
		"message": "Data source deleted successfully",
	}))
}

// ============================================================================
// UTILITY OPERATIONS
// ============================================================================

func (dsh *DataSourceHandler) CheckDataSourceExists(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	exists, err := dsh.dataSourceService.CheckDataSourceExists(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("CHECK_FAILED", err.Error()))
	}

	response := map[string]any{
		"data_source_id": id,
		"exists":         exists,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (dsh *DataSourceHandler) GetDataSourceCount(c fiber.Ctx) error {
	count, err := dsh.dataSourceService.GetDataSourceCount()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	response := map[string]any{
		"total_count": count,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (dsh *DataSourceHandler) GetActiveDataSourceCount(c fiber.Ctx) error {
	count, err := dsh.dataSourceService.GetActiveDataSourceCount()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	response := map[string]any{
		"active_count": count,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (dsh *DataSourceHandler) GetDataSourceCountByType(c fiber.Ctx) error {
	typeParam := c.Params("type")
	dataSourceType := models.DataSourceType(typeParam)

	count, err := dsh.dataSourceService.GetDataSourceCountByType(dataSourceType)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	response := map[string]any{
		"data_source_type": dataSourceType,
		"count":            count,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (dsh *DataSourceHandler) GetDataSourceCountByTier(c fiber.Ctx) error {
	tierIdParam := c.Params("tierId")
	tierId, err := uuid.Parse(tierIdParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_ID", "Invalid UUID format"))
	}

	count, err := dsh.dataSourceService.GetDataSourceCountByTier(tierId)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	response := map[string]any{
		"tier_id": tierId,
		"count":   count,
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}
