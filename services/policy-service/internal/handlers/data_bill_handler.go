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
	"github.com/robfig/cron/v3"
)

type DataBillHandler struct {
	basePolicyService  *services.BasePolicyService
	notificationHelper *event.NotificationHelper
}

func NewDataBillHandler(basePolicyService *services.BasePolicyService, notificationHelper *event.NotificationHelper) *DataBillHandler {
	handler := &DataBillHandler{
		basePolicyService:  basePolicyService,
		notificationHelper: notificationHelper,
	}
	handler.startCron()
	return handler
}

func (h *DataBillHandler) startCron() {
	c := cron.New()
	c.AddFunc("@monthly", func() {
		err := h.MarkPoliciesForPayment(context.Background())
		if err != nil {
			// Log error, but since no logger, just ignore or print
		}
	})
	c.Start()
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
	insuranceProviderId := c.Get("x-user-id")
	activePolicies, err := h.basePolicyService.GetActivePolicies(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrieve active policies"))
	}

	filtered := []models.BasePolicy{}
	for _, p := range activePolicies {
		if p.InsuranceProviderID == insuranceProviderId && time.Since(p.CreatedAt) >= 30*24*time.Hour {
			filtered = append(filtered, p)
		}
	}

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

func (h *DataBillHandler) Register(app *fiber.App) {
	app.Get("/data-bill", h.GetDataBillHandler)
	app.Post("/data-bill/:id/mark-payment", h.MarkPolicyForPaymentManual)
}
