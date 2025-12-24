package handlers

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"policy-service/internal/event"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type DataBillHandler struct {
	basePolicyService       *services.BasePolicyService
	notificationHelper      *event.NotificationHelper
	registeredPolicyService *services.RegisteredPolicyService
}

func NewDataBillHandler(basePolicyService *services.BasePolicyService, notificationHelper *event.NotificationHelper, registeredPolicyService *services.RegisteredPolicyService) *DataBillHandler {
	handler := &DataBillHandler{
		basePolicyService:       basePolicyService,
		notificationHelper:      notificationHelper,
		registeredPolicyService: registeredPolicyService,
	}
	return handler
}

func (h *DataBillHandler) MarkPoliciesForPayment(ctx context.Context) error {
	activePolicies, err := h.basePolicyService.GetActivePolicies(ctx)
	if err != nil {
		return err
	}

	for _, policy := range activePolicies {
		if time.Since(policy.CreatedAt) >= 30*24*time.Hour {
			// Update status to payment_due
			err := h.basePolicyService.UpdateBasePolicyStatus(ctx, policy.ID, models.BasePolicyPaymentDue)
			if err != nil {
				fmt.Printf("Error updating status for policy %s: %v\n", policy.ID, err)
				continue
			}

			// Send notification to insurance provider
			err = h.notificationHelper.NotifyMultipleUsers(ctx, "Payment Due", fmt.Sprintf("Policy %s is due for payment", policy.ID), []string{policy.InsuranceProviderID})
			if err != nil {
				fmt.Printf("Error sending notification for policy %s: %v\n", policy.ID, err)
			} else {
				fmt.Printf("Notification sent for policy %s\n", policy.ID)
			}
		}
	}

	return nil
}

func (h *DataBillHandler) GetDataBillHandler(c fiber.Ctx) error {
	token := c.Get("Authorization")
	token = token[len("Bearer "):]
	insuranceProfile, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to get insurance partner profile"))
	}

	insuranceProviderId, err := h.registeredPolicyService.GetPartnerID(insuranceProfile)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to extract partner ID"))
	}

	fmt.Printf("Insurance Provider ID: %s\n", insuranceProviderId)
	paymentDuePolicies, err := h.basePolicyService.GetPaymentDuePolicies(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrieve payment due policies"))
	}
	fmt.Printf("Total payment due policies: %d\n", len(paymentDuePolicies))

	filtered := []models.BasePolicy{}
	for _, p := range paymentDuePolicies {
		fmt.Printf("Policy ID: %s, Provider ID: %s, Status: %s\n", p.ID, p.InsuranceProviderID, p.Status)
		if p.InsuranceProviderID == insuranceProviderId {
			filtered = append(filtered, p)
		}
	}
	fmt.Printf("Filtered policies: %d\n", len(filtered))

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(filtered))
}

func (h *DataBillHandler) MarkPolicyForPaymentManual(c fiber.Ctx) error {
	policyID := c.Params("id")
	if policyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "policy ID is required"))
	}

	// Parse UUID
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "invalid policy ID"))
	}

	// Update status to payment_due
	err = h.basePolicyService.UpdateBasePolicyStatus(c.Context(), policyUUID, models.BasePolicyPaymentDue)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("UPDATE_ERROR", "failed to update policy status"))
	}

	// Send notification for manual mark
	err = h.notificationHelper.NotifyMultipleUsers(c.Context(), "Manual Payment Mark", fmt.Sprintf("Policy %s manually marked for payment", policyID), []string{"admin"}) // or specific user
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("NOTIFICATION_ERROR", "failed to send notification"))
	}

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse("Policy marked for payment"))
}

func (h *DataBillHandler) getDataCost(policyId uuid.UUID) (float64, error) {
	policies, err := h.registeredPolicyService.GetByBasePolicy(context.Background(), policyId)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, p := range policies {
		total += p.TotalDataCost
	}
	return total, nil
}

func (h *DataBillHandler) GetDataCost(c fiber.Ctx) error {
	policyID := c.Params("id")
	if policyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "policy ID is required"))
	}

	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "invalid policy ID"))
	}

	totalCost, err := h.getDataCost(policyUUID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to calculate data cost"))
	}

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(totalCost))
}

func (h *DataBillHandler) Register(app *fiber.App) {
	app.Get("policy/protected/api/v2/data-bill", h.GetDataBillHandler)
	app.Get("policy/protected/api/v2/data-bill/cost/:id", h.GetDataCost)
	app.Post("policy/protected/api/v2/data-bill/mark-payment/:id", h.MarkPolicyForPaymentManual)
}
