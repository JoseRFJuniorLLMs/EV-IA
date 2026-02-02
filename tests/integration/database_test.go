package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDatabase_UserCRUD tests user database operations
func TestDatabase_UserCRUD(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.DB == nil {
		t.Skip("Database not available")
	}

	SetupSchema(t, env.DB)
	CleanDatabase(t, env.DB)

	ctx := context.Background()
	userID := uuid.New().String()

	// Create user
	t.Run("CreateUser", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, userID, "Test User", "test@example.com", "hashed_password", "user", "Active", time.Now(), time.Now())

		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	})

	// Read user
	t.Run("ReadUser", func(t *testing.T) {
		var id, name, email string
		err := env.DB.QueryRowContext(ctx, `
			SELECT id, name, email FROM users WHERE id = $1
		`, userID).Scan(&id, &name, &email)

		if err != nil {
			t.Fatalf("Failed to read user: %v", err)
		}

		if name != "Test User" {
			t.Errorf("Expected name 'Test User', got '%s'", name)
		}

		if email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", email)
		}
	})

	// Update user
	t.Run("UpdateUser", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			UPDATE users SET name = $1, updated_at = $2 WHERE id = $3
		`, "Updated User", time.Now(), userID)

		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		var name string
		env.DB.QueryRowContext(ctx, `SELECT name FROM users WHERE id = $1`, userID).Scan(&name)

		if name != "Updated User" {
			t.Errorf("Expected name 'Updated User', got '%s'", name)
		}
	})

	// Delete user
	t.Run("DeleteUser", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		var count int
		env.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE id = $1`, userID).Scan(&count)

		if count != 0 {
			t.Error("User should have been deleted")
		}
	})
}

// TestDatabase_ChargePointCRUD tests charge point database operations
func TestDatabase_ChargePointCRUD(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.DB == nil {
		t.Skip("Database not available")
	}

	SetupSchema(t, env.DB)
	CleanDatabase(t, env.DB)

	ctx := context.Background()
	cpID := "CP001"

	// Create charge point
	t.Run("CreateChargePoint", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			INSERT INTO charge_points (id, vendor, model, status, latitude, longitude, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, cpID, "ABB", "Terra 184", "Available", -23.55, -46.63, time.Now(), time.Now())

		if err != nil {
			t.Fatalf("Failed to create charge point: %v", err)
		}
	})

	// Read charge point
	t.Run("ReadChargePoint", func(t *testing.T) {
		var id, vendor, model, status string
		err := env.DB.QueryRowContext(ctx, `
			SELECT id, vendor, model, status FROM charge_points WHERE id = $1
		`, cpID).Scan(&id, &vendor, &model, &status)

		if err != nil {
			t.Fatalf("Failed to read charge point: %v", err)
		}

		if vendor != "ABB" {
			t.Errorf("Expected vendor 'ABB', got '%s'", vendor)
		}

		if status != "Available" {
			t.Errorf("Expected status 'Available', got '%s'", status)
		}
	})

	// Update status
	t.Run("UpdateChargePointStatus", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			UPDATE charge_points SET status = $1, updated_at = $2 WHERE id = $3
		`, "Occupied", time.Now(), cpID)

		if err != nil {
			t.Fatalf("Failed to update charge point: %v", err)
		}

		var status string
		env.DB.QueryRowContext(ctx, `SELECT status FROM charge_points WHERE id = $1`, cpID).Scan(&status)

		if status != "Occupied" {
			t.Errorf("Expected status 'Occupied', got '%s'", status)
		}
	})

	// Find nearby (geo query)
	t.Run("FindNearby", func(t *testing.T) {
		// This is a simplified test - real implementation would use PostGIS
		rows, err := env.DB.QueryContext(ctx, `
			SELECT id FROM charge_points
			WHERE latitude BETWEEN $1 AND $2
			AND longitude BETWEEN $3 AND $4
		`, -24.0, -23.0, -47.0, -46.0)

		if err != nil {
			t.Fatalf("Failed to find nearby: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}

		if count == 0 {
			t.Error("Expected to find at least one charge point")
		}
	})
}

// TestDatabase_TransactionCRUD tests transaction database operations
func TestDatabase_TransactionCRUD(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.DB == nil {
		t.Skip("Database not available")
	}

	SetupSchema(t, env.DB)
	CleanDatabase(t, env.DB)

	ctx := context.Background()

	// Create prerequisites
	userID := uuid.New().String()
	cpID := "CP001"
	txID := uuid.New().String()

	// Setup user and charge point
	env.DB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
		VALUES ($1, 'Test', 'test@test.com', 'pass', 'user', 'Active', $2, $2)
	`, userID, time.Now())

	env.DB.ExecContext(ctx, `
		INSERT INTO charge_points (id, vendor, model, status, created_at, updated_at)
		VALUES ($1, 'ABB', 'Terra', 'Available', $2, $2)
	`, cpID, time.Now())

	// Create transaction
	t.Run("CreateTransaction", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			INSERT INTO transactions (id, charge_point_id, connector_id, user_id, status, meter_start, start_time, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, txID, cpID, 1, userID, "Active", 1000.0, time.Now(), time.Now())

		if err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}
	})

	// Read active transaction
	t.Run("ReadActiveTransaction", func(t *testing.T) {
		var id, status string
		err := env.DB.QueryRowContext(ctx, `
			SELECT id, status FROM transactions WHERE user_id = $1 AND status = 'Active'
		`, userID).Scan(&id, &status)

		if err != nil {
			t.Fatalf("Failed to read active transaction: %v", err)
		}

		if id != txID {
			t.Errorf("Expected transaction ID '%s', got '%s'", txID, id)
		}
	})

	// Complete transaction
	t.Run("CompleteTransaction", func(t *testing.T) {
		endTime := time.Now()
		_, err := env.DB.ExecContext(ctx, `
			UPDATE transactions SET status = 'Completed', meter_stop = $1, end_time = $2, updated_at = $3
			WHERE id = $4
		`, 1025.5, endTime, endTime, txID)

		if err != nil {
			t.Fatalf("Failed to complete transaction: %v", err)
		}

		var meterStop float64
		var status string
		env.DB.QueryRowContext(ctx, `
			SELECT meter_stop, status FROM transactions WHERE id = $1
		`, txID).Scan(&meterStop, &status)

		if meterStop != 1025.5 {
			t.Errorf("Expected meter_stop 1025.5, got %f", meterStop)
		}

		if status != "Completed" {
			t.Errorf("Expected status 'Completed', got '%s'", status)
		}
	})

	// Get transaction history
	t.Run("GetTransactionHistory", func(t *testing.T) {
		rows, err := env.DB.QueryContext(ctx, `
			SELECT id, status, meter_start, meter_stop
			FROM transactions
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)

		if err != nil {
			t.Fatalf("Failed to get history: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}

		if count == 0 {
			t.Error("Expected at least one transaction in history")
		}
	})
}

// TestDatabase_WalletOperations tests wallet database operations
func TestDatabase_WalletOperations(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.DB == nil {
		t.Skip("Database not available")
	}

	SetupSchema(t, env.DB)
	CleanDatabase(t, env.DB)

	ctx := context.Background()

	// Create user
	userID := uuid.New().String()
	walletID := uuid.New().String()

	env.DB.ExecContext(ctx, `
		INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
		VALUES ($1, 'Test', 'test@test.com', 'pass', 'user', 'Active', $2, $2)
	`, userID, time.Now())

	// Create wallet
	t.Run("CreateWallet", func(t *testing.T) {
		_, err := env.DB.ExecContext(ctx, `
			INSERT INTO wallets (id, user_id, balance, currency, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`, walletID, userID, 0.0, "BRL", time.Now())

		if err != nil {
			t.Fatalf("Failed to create wallet: %v", err)
		}
	})

	// Add funds
	t.Run("AddFunds", func(t *testing.T) {
		tx, err := env.DB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Update balance
		_, err = tx.ExecContext(ctx, `
			UPDATE wallets SET balance = balance + $1, updated_at = $2 WHERE id = $3
		`, 100.0, time.Now(), walletID)

		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to add funds: %v", err)
		}

		// Record transaction
		_, err = tx.ExecContext(ctx, `
			INSERT INTO wallet_transactions (id, wallet_id, user_id, type, amount, balance, description, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, uuid.New().String(), walletID, userID, "credit", 100.0, 100.0, "Test deposit", time.Now())

		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to record transaction: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify balance
		var balance float64
		env.DB.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&balance)

		if balance != 100.0 {
			t.Errorf("Expected balance 100.0, got %f", balance)
		}
	})

	// Deduct funds
	t.Run("DeductFunds", func(t *testing.T) {
		// Get current balance
		var currentBalance float64
		env.DB.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&currentBalance)

		if currentBalance < 30.0 {
			t.Skip("Insufficient balance for test")
		}

		_, err := env.DB.ExecContext(ctx, `
			UPDATE wallets SET balance = balance - $1, updated_at = $2 WHERE id = $3 AND balance >= $1
		`, 30.0, time.Now(), walletID)

		if err != nil {
			t.Fatalf("Failed to deduct funds: %v", err)
		}

		var newBalance float64
		env.DB.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&newBalance)

		if newBalance != 70.0 {
			t.Errorf("Expected balance 70.0, got %f", newBalance)
		}
	})

	// Insufficient balance check
	t.Run("InsufficientBalance", func(t *testing.T) {
		result, err := env.DB.ExecContext(ctx, `
			UPDATE wallets SET balance = balance - $1, updated_at = $2 WHERE id = $3 AND balance >= $1
		`, 1000.0, time.Now(), walletID)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected != 0 {
			t.Error("Should not have deducted funds due to insufficient balance")
		}
	})
}

// TestDatabase_Transactions tests database transactions (ACID)
func TestDatabase_Transactions(t *testing.T) {
	env := SetupTestEnvironment(t)
	if env == nil || env.DB == nil {
		t.Skip("Database not available")
	}

	SetupSchema(t, env.DB)
	CleanDatabase(t, env.DB)

	ctx := context.Background()

	// Test rollback
	t.Run("Rollback", func(t *testing.T) {
		tx, err := env.DB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		userID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
			VALUES ($1, 'Rollback Test', 'rollback@test.com', 'pass', 'user', 'Active', $2, $2)
		`, userID, time.Now())

		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}

		// Rollback
		if err := tx.Rollback(); err != nil {
			t.Fatalf("Failed to rollback: %v", err)
		}

		// Verify user doesn't exist
		var count int
		env.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE id = $1`, userID).Scan(&count)

		if count != 0 {
			t.Error("User should not exist after rollback")
		}
	})

	// Test commit
	t.Run("Commit", func(t *testing.T) {
		tx, err := env.DB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		userID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
			VALUES ($1, 'Commit Test', 'commit@test.com', 'pass', 'user', 'Active', $2, $2)
		`, userID, time.Now())

		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify user exists
		var count int
		env.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE id = $1`, userID).Scan(&count)

		if count != 1 {
			t.Error("User should exist after commit")
		}
	})
}

// skipIfNoDatabase skips the test if database is not available
func skipIfNoDatabase(t *testing.T, db *sql.DB) {
	if db == nil {
		t.Skip("Database not available")
	}
}
