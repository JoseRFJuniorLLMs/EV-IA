package handlers

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/ports"
)

type TransactionHandler struct {
	service ports.TransactionService
	log     *zap.Logger
}

func NewTransactionHandler(service ports.TransactionService, log *zap.Logger) *TransactionHandler {
	return &TransactionHandler{
		service: service,
		log:     log,
	}
}

type StartTransactionRequest struct {
	DeviceID    string `json:"device_id"`
	ConnectorID int    `json:"connector_id"`
	IdTag       string `json:"rfid_tag"` // Optional
}

func (h *TransactionHandler) Start(c *fiber.Ctx) error {
	var req StartTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	userID := c.Locals("user_id").(string) // Assumes middleware sets this

	tx, err := h.service.StartTransaction(c.Context(), req.DeviceID, req.ConnectorID, userID, req.IdTag)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tx)
}

func (h *TransactionHandler) Stop(c *fiber.Ctx) error {
	id := c.Params("id")
	tx, err := h.service.StopTransaction(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tx)
}

func (h *TransactionHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	tx, err := h.service.GetTransaction(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if tx == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Transaction not found"})
	}
	return c.JSON(tx)
}

func (h *TransactionHandler) GetHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	txs, err := h.service.GetTransactionHistory(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(txs)
}

func (h *TransactionHandler) GetActive(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	tx, err := h.service.GetActiveTransaction(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if tx == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No active transaction"})
	}
	return c.JSON(tx)
}
