package handlers

import (
	utils "agrisa_utils"
	"net/http"
	"policy-service/internal/services"

	"github.com/gofiber/fiber/v3"
)

type BasePolicyTriggerHandler struct {
	BasePolicyTriggerService *services.BasePolicyTriggerService
}

func NewBasePolicyTriggerHandler(basePolicyTriggerService *services.BasePolicyTriggerService) *BasePolicyTriggerHandler {
	return &BasePolicyTriggerHandler{
		BasePolicyTriggerService: basePolicyTriggerService,
	}
}

func (bph *BasePolicyTriggerHandler) Register(app *fiber.App) {
	publicGR := app.Group("policy/public/api/v2")
	// protectedGR := app.Group("policy/protected/api/v2")

	publicGR.Get("/base-policy-triggers/:id", bph.GetBasePolicyTriggersByID)
}

func (bph *BasePolicyTriggerHandler) GetBasePolicyTriggersByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Base Policy Trigger ID is required"))
	}

	conditions := []utils.Condition{
		{
			Field:    "id",
			Operator: "=",
			Value:    id,
		},
	}

	orderBy := []string{}

	basePolicyTriggers, err := bph.BasePolicyTriggerService.GetBasePolicyTriggersByFilter(conditions, orderBy)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "Failed to get Base Policy Trigger"))
	}
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(basePolicyTriggers))
}
