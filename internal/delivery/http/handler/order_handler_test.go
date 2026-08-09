package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"flashsale-go/internal/delivery/http/handler"
	"flashsale-go/internal/domain"
	"flashsale-go/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrderUsecase struct {
	mock.Mock
}

func (m *MockOrderUsecase) CreateFlashSaleOrder(ctx context.Context, userID string, dto *domain.CreateOrderDTO) (*domain.OrderResponseDTO, error) {
	args := m.Called(ctx, userID, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OrderResponseDTO), args.Error(1)
}

func (m *MockOrderUsecase) ProcessOrderFromQueue(ctx context.Context, event *domain.OrderEventPayload) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOrderUsecase) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderUsecase) SubscribeOrderStatusStream(ctx context.Context, orderID string) (<-chan string, func(), error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, func() {}, args.Error(2)
	}
	return args.Get(0).(<-chan string), args.Get(1).(func()), args.Error(2)
}

func TestOrderHandler_CreateFlashSaleOrder_Unauthorized(t *testing.T) {
	mockUsecase := new(MockOrderUsecase)
	orderHandler := handler.NewOrderHandler(mockUsecase)

	app := fiber.New()
	app.Post("/orders/flash-sale", orderHandler.CreateFlashSaleOrder)

	req := httptest.NewRequest("POST", "/orders/flash-sale", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestOrderHandler_CreateFlashSaleOrder_Success(t *testing.T) {
	mockUsecase := new(MockOrderUsecase)
	orderHandler := handler.NewOrderHandler(mockUsecase)

	app := fiber.New()
	app.Post("/orders/flash-sale", func(c *fiber.Ctx) error {
		c.Locals("userID", "user-test-123")
		return orderHandler.CreateFlashSaleOrder(c)
	})

	dto := domain.CreateOrderDTO{
		ProductID: "prod-123",
		Quantity:  1,
	}
	body, _ := json.Marshal(dto)

	mockUsecase.On("CreateFlashSaleOrder", mock.Anything, "user-test-123", &dto).Return(&domain.OrderResponseDTO{
		OrderID: "ORD-999",
		Status:  domain.OrderStatusPending,
		Message: "Order received",
	}, nil)

	req := httptest.NewRequest("POST", "/orders/flash-sale", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)
	mockUsecase.AssertExpectations(t)
}

func TestOrderHandler_CreateFlashSaleOrder_OutOfStock(t *testing.T) {
	mockUsecase := new(MockOrderUsecase)
	orderHandler := handler.NewOrderHandler(mockUsecase)

	app := fiber.New()
	app.Post("/orders/flash-sale", func(c *fiber.Ctx) error {
		c.Locals("userID", "user-test-123")
		return orderHandler.CreateFlashSaleOrder(c)
	})

	dto := domain.CreateOrderDTO{
		ProductID: "prod-123",
		Quantity:  1,
	}
	body, _ := json.Marshal(dto)

	mockUsecase.On("CreateFlashSaleOrder", mock.Anything, "user-test-123", &dto).Return(nil, usecase.ErrOutOfStock)

	req := httptest.NewRequest("POST", "/orders/flash-sale", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestOrderHandler_GetOrderByID_Success(t *testing.T) {
	mockUsecase := new(MockOrderUsecase)
	orderHandler := handler.NewOrderHandler(mockUsecase)

	app := fiber.New()
	app.Get("/orders/:id", func(c *fiber.Ctx) error {
		c.Locals("userID", "user-123")
		c.Locals("userRole", "customer")
		return orderHandler.GetOrderByID(c)
	})

	mockUsecase.On("GetOrderByID", mock.Anything, "ORD-123").Return(&domain.Order{
		OrderID:   "ORD-123",
		UserID:    "user-123",
		ProductID: "prod-123",
		Status:    domain.OrderStatusCompleted,
	}, nil)

	req := httptest.NewRequest("GET", "/orders/ORD-123", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestOrderHandler_StreamOrderStatus_HidesOtherUsersOrder(t *testing.T) {
	mockUsecase := new(MockOrderUsecase)
	orderHandler := handler.NewOrderHandler(mockUsecase)
	app := fiber.New()
	app.Get("/orders/:id/stream", func(c *fiber.Ctx) error {
		c.Locals("userID", "user-other")
		c.Locals("userRole", "customer")
		return orderHandler.StreamOrderStatus(c)
	})

	mockUsecase.On("GetOrderByID", mock.Anything, "ORD-private").Return(&domain.Order{
		OrderID: "ORD-private",
		UserID:  "user-owner",
	}, nil)

	req := httptest.NewRequest("GET", "/orders/ORD-private/stream", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	mockUsecase.AssertNotCalled(t, "SubscribeOrderStatusStream", mock.Anything, mock.Anything)
}
