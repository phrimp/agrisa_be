package handlers

import (
	"notification-service/internal/google"

	"github.com/gofiber/fiber/v3"
)

type EmailHandler struct {
	emailService *google.EmailService
}

func NewEmailHandler(emailService *google.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

func (e *EmailHandler) Register(app *fiber.App) {
	protectedGr := app.Group("/notification/protected/api/v2")
	emailGr := protectedGr.Group("/email")

	emailGr.Post("/send/greet", e.Greet)
}

func (e *EmailHandler) Greet(c fiber.Ctx) error {
	type GreetRequest struct {
		To   string `json:"to"`
		Name string `json:"name"`
	}
	var greetRequest GreetRequest

	if err := c.Bind().Body(&greetRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if err := e.emailService.GreetingEmail(greetRequest.To, greetRequest.Name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "Failed to send email",
			"detail": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).SendString("Greeting sent")
}
