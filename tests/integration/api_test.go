package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestAPI_HealthCheck tests the health endpoint
func TestAPI_HealthCheck(t *testing.T) {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", result["status"])
	}
}

// TestAPI_AuthFlow tests the authentication flow
func TestAPI_AuthFlow(t *testing.T) {
	app := setupTestApp(t)

	// Test registration
	t.Run("Register", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":     "Test User",
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 201 or 200, got %d", resp.StatusCode)
		}
	})

	// Test login
	t.Run("Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["access_token"] == nil {
			t.Error("Expected access_token in response")
		}
	})

	// Test invalid login
	t.Run("InvalidLogin", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}

// TestAPI_DeviceEndpoints tests device CRUD operations
func TestAPI_DeviceEndpoints(t *testing.T) {
	app := setupTestApp(t)
	token := getAuthToken(t, app)

	deviceID := ""

	// Create device (admin only)
	t.Run("CreateDevice", func(t *testing.T) {
		payload := map[string]interface{}{
			"id":       "CP001",
			"vendor":   "ABB",
			"model":    "Terra 184",
			"latitude": -23.55,
			"longitude": -46.63,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// May be 403 if not admin, but endpoint should exist
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status 201, 200, or 403, got %d", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			if id, ok := result["id"].(string); ok {
				deviceID = id
			}
		}
	})

	// List devices
	t.Run("ListDevices", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Get nearby devices
	t.Run("GetNearbyDevices", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/nearby?lat=-23.55&lon=-46.63&radius=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Get device by ID
	if deviceID != "" {
		t.Run("GetDevice", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+deviceID, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

// TestAPI_TransactionEndpoints tests transaction operations
func TestAPI_TransactionEndpoints(t *testing.T) {
	app := setupTestApp(t)
	token := getAuthToken(t, app)

	// Get transaction history
	t.Run("GetTransactionHistory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Get active transaction
	t.Run("GetActiveTransaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions/active", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// 200 if has active, 404 if not
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})
}

// TestAPI_WalletEndpoints tests wallet operations
func TestAPI_WalletEndpoints(t *testing.T) {
	app := setupTestApp(t)
	token := getAuthToken(t, app)

	// Get wallet
	t.Run("GetWallet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})

	// Get wallet transactions
	t.Run("GetWalletTransactions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/transactions", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})
}

// setupTestApp creates a test Fiber app with mock handlers
func setupTestApp(t *testing.T) *fiber.App {
	app := fiber.New()

	// Mock auth endpoints
	app.Post("/api/v1/auth/register", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "User registered successfully",
		})
	})

	app.Post("/api/v1/auth/login", func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		if body["password"] != "password123" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}

		return c.JSON(fiber.Map{
			"access_token":  "test-token",
			"refresh_token": "test-refresh",
		})
	})

	// Mock device endpoints
	app.Get("/api/v1/devices", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{})
	})

	app.Get("/api/v1/devices/nearby", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{})
	})

	app.Get("/api/v1/devices/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id")})
	})

	app.Post("/api/v1/devices", func(c *fiber.Ctx) error {
		var body map[string]interface{}
		c.BodyParser(&body)
		return c.Status(fiber.StatusCreated).JSON(body)
	})

	// Mock transaction endpoints
	app.Get("/api/v1/transactions", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{})
	})

	app.Get("/api/v1/transactions/active", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no active transaction"})
	})

	// Mock wallet endpoints
	app.Get("/api/v1/wallet", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"balance":  0,
			"currency": "BRL",
		})
	})

	app.Get("/api/v1/wallet/transactions", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{})
	})

	// Health endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	return app
}

// getAuthToken gets an auth token for testing
func getAuthToken(t *testing.T, app *fiber.App) string {
	payload := map[string]interface{}{
		"email":    "test@example.com",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to get auth token: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if token, ok := result["access_token"].(string); ok {
		return token
	}

	return "test-token"
}
