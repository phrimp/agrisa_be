package handlers

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type CancelRequestHandler struct {
	registeredPolicyService *services.RegisteredPolicyService
	cancelRequestService    *services.CancelRequestService
}

func NewCancelRequestHandler(registeredPolicyService *services.RegisteredPolicyService, cancelRequestHandler *services.CancelRequestService) *CancelRequestHandler {
	return &CancelRequestHandler{
		registeredPolicyService: registeredPolicyService,
		cancelRequestService:    cancelRequestHandler,
	}
}

func (h *CancelRequestHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Claim routes
	cancelRequestGr := protectedGr.Group("/cancel_request")
	cancelRequestGr.Post("/", h.CreateNewRequest)
	cancelRequestGr.Put("/review/:id", h.ReviewCancelRequest)
	cancelRequestGr.Put("/resolve-dispute/:id", h.ResolveDispute)
	cancelRequestGr.Put("/compensation-amount/:id", h.GetCompensationAmount)
	cancelRequestGr.Post("/revoke/:id", h.RevokeRequest)

	farmerGr := cancelRequestGr.Group("/read-own")
	farmerGr.Get("/me", h.GetAllMyRequests)

	partnerGroup := cancelRequestGr.Group("/read-partner")
	partnerGroup.Get("/own", h.GetAllPartnerRequest)
}

func (h *CancelRequestHandler) GetAllMyRequests(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}
	requests, err := h.cancelRequestService.GetAllFarmerCancelRequest(c.Context(), userID)
	if err != nil {
		slog.Error("Failed to get farmer requests", "farmer_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve requests"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"claims":    requests,
		"count":     len(requests),
		"farmer_id": userID,
	}))
}

func (h *CancelRequestHandler) ReviewCancelRequest(c fiber.Ctx) error {
	var req models.ReviewCancelRequestReq
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body: "+err.Error()))
	}

	requestIDStr := c.Params("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid cancel request ID format"))
	}

	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}
	requestBy := userID
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	partnerFound := true
	if err != nil {
		if !strings.Contains(err.Error(), "insurance partner profile not found") {
			slog.Error("error retriving partner profile", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
		}
		partnerFound = false
	}

	if partnerFound {
		requestBy, err = h.registeredPolicyService.GetPartnerID(partnerProfileData)
		if err != nil {
			slog.Error("error retriving partner id", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
		}
	}
	req.ReviewedBy = requestBy
	req.RequestID = requestID
	res, err := h.cancelRequestService.ReviewCancelRequest(c.Context(), req)
	if err != nil {
		slog.Error("error reviewing cancel request", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", fmt.Sprintf("Failed to reviewing cancel request: %s", err)))
	}

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(res))
}

func (h *CancelRequestHandler) ResolveDispute(c fiber.Ctx) error {
	var req models.ResolveConflictCancelRequestReq
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body: "+err.Error()))
	}

	requestIDStr := c.Params("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid cancel request ID format"))
	}

	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}
	requestBy := userID
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	partnerFound := true
	if err != nil {
		if !strings.Contains(err.Error(), "insurance partner profile not found") {
			slog.Error("error retriving partner profile", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
		}
		partnerFound = false
	}

	if partnerFound {
		requestBy, err = h.registeredPolicyService.GetPartnerID(partnerProfileData)
		if err != nil {
			slog.Error("error retriving partner id", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
		}
	}
	req.ReviewedBy = requestBy
	req.RequestID = requestID
	res, err := h.cancelRequestService.ResolveConflict(c.Context(), req)
	if err != nil {
		slog.Error("error reviewing cancel request", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to reviewing cancel request"))
	}

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(res))
}

func (h *CancelRequestHandler) GetCompensationAmount(c fiber.Ctx) error {
	requestIDStr := c.Params("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid cancel request ID format"))
	}

	policyIDStr := c.Query("policy_id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid cancel request ID format"))
	}
	compensationAmount, err := h.cancelRequestService.GetCompensationAmount(c.Context(), requestID, policyID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retriving request compensation amount"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(compensationAmount))
}

func (h *CancelRequestHandler) RevokeRequest(c fiber.Ctx) error {
	requestIDStr := c.Params("id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid cancel request ID format"))
	}
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}
	requestBy := userID
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	partnerFound := true
	if err != nil {
		if !strings.Contains(err.Error(), "insurance partner profile not found") {
			slog.Error("error retriving partner profile", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
		}
		partnerFound = false
	}

	if partnerFound {
		requestBy, err = h.registeredPolicyService.GetPartnerID(partnerProfileData)
		if err != nil {
			slog.Error("error retriving partner id", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
		}
	}

	err = h.cancelRequestService.RevokeRequest(c.Context(), requestID, requestBy)
	if err != nil {
		slog.Error("revoke cancel request failed", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("INTERNAL", "Revoke request failed"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse("cancel request revoked"))
}

func (h *CancelRequestHandler) CreateNewRequest(c fiber.Ctx) error {
	var req models.CreateCancelRequestRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body: "+err.Error()))
	}

	policyIDStr := c.Query("policy_id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}
	requestBy := userID
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	partnerFound := true
	if err != nil {
		if !strings.Contains(err.Error(), "insurance partner profile not found") {
			slog.Error("error retriving partner profile", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
		}
		partnerFound = false
	}

	if partnerFound {
		requestBy, err = h.registeredPolicyService.GetPartnerID(partnerProfileData)
		if err != nil {
			slog.Error("error retriving partner id", "error", err)
			return c.Status(http.StatusInternalServerError).JSON(
				utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
		}
	}

	res, err := h.cancelRequestService.CreateCancelRequest(c.Context(), policyID, requestBy, req)
	if err != nil {
		slog.Error("error creating cancel request", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("CREATE_FAILED", err.Error()))
	}
	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(res))
}

func (h *CancelRequestHandler) GetAllPartnerRequest(c fiber.Ctx) error {
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerID, err := h.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	providerID := partnerID
	requests, err := h.cancelRequestService.GetAllProviderCancelRequests(c.Context(), providerID)
	if err != nil {
		slog.Error("Failed to get farmer requests", "provider_id", providerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve requests"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"claims":      requests,
		"count":       len(requests),
		"provider_id": providerID,
	}))
}
