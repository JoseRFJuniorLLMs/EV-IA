package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/seu-repo/sigec-ve/internal/domain"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type DeviceHandler struct {
	service ports.DeviceService
	log     *zap.Logger
}

func NewDeviceHandler(service ports.DeviceService, log *zap.Logger) *DeviceHandler {
	return &DeviceHandler{
		service: service,
		log:     log,
	}
}

func (h *DeviceHandler) List(c *fiber.Ctx) error {
	filter := make(map[string]interface{})
	// Populate filter from query params
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	devices, err := h.service.ListDevices(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(devices)
}

func (h *DeviceHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	device, err := h.service.GetDevice(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if device == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Device not found"})
	}
	return c.JSON(device)
}

func (h *DeviceHandler) GetNearby(c *fiber.Ctx) error {
	lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
	lon, _ := strconv.ParseFloat(c.Query("lon"), 64)
	radius, _ := strconv.ParseFloat(c.Query("radius"), 64)

	devices, err := h.service.GetNearby(c.Context(), lat, lon, radius)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(devices)
}

func (h *DeviceHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Status domain.ChargePointStatus `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	if err := h.service.UpdateStatus(c.Context(), id, req.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusOK)
}
