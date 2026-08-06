package handler

import (
	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"
	"flashsale-go/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type OrderHandler struct {
	orderUsecase usecase.OrderUsecase
}

func NewOrderHandler(orderUsecase usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{
		orderUsecase: orderUsecase,
	}
}

// CreateFlashSaleOrder handles high-concurrency order placement
func (h *OrderHandler) CreateFlashSaleOrder(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return utils.JSONError(c, fiber.StatusUnauthorized, "Unauthorized user", "")
	}

	var dto domain.CreateOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.orderUsecase.CreateFlashSaleOrder(c.Context(), userID, &dto)
	if err != nil {
		if err.Error() == "product out of stock" {
			return utils.JSONError(c, fiber.StatusBadRequest, "Out of stock", "The flash sale item has sold out")
		}
		if err.Error() == "order processing in progress, please do not double click" {
			return utils.JSONError(c, fiber.StatusTooManyRequests, "Duplicate request", err.Error())
		}
		return utils.JSONError(c, fiber.StatusInternalServerError, "Failed to place flash sale order", err.Error())
	}

	// 202 Accepted return status
	return utils.JSONSuccess(c, fiber.StatusAccepted, "Order placed successfully in queue", res)
}

// GetOrderByID retrieves order details
func (h *OrderHandler) GetOrderByID(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return utils.JSONError(c, fiber.StatusBadRequest, "Missing order ID parameter", "")
	}

	order, err := h.orderUsecase.GetOrderByID(c.Context(), orderID)
	if err != nil {
		return utils.JSONError(c, fiber.StatusNotFound, "Order not found", err.Error())
	}

	return utils.JSONSuccess(c, fiber.StatusOK, "Order details fetched successfully", order)
}
