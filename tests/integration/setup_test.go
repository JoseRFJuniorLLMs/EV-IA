package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

// TestEnv holds test environment resources
type TestEnv struct {
	DB              *sql.DB
	Redis           *redis.Client
	PostgresContainer testcontainers.Container
	RedisContainer   testcontainers.Container
	Logger          *zap.Logger
	ctx             context.Context
}

var testEnv *TestEnv

// SetupTestEnvironment initializes the test environment with containers
func SetupTestEnvironment(t *testing.T) *TestEnv {
	if testEnv != nil {
		return testEnv
	}

	ctx := context.Background()

	// Check if using external services (CI environment)
	if os.Getenv("DATABASE_URL") != "" {
		return setupExternalServices(t, ctx)
	}

	// Use testcontainers for local testing
	return setupContainers(t, ctx)
}

func setupExternalServices(t *testing.T, ctx context.Context) *TestEnv {
	logger, _ := zap.NewDevelopment()

	// Connect to external Postgres
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Connect to external Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("Failed to parse Redis URL: %v", err)
	}

	redisClient := redis.NewClient(opt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	testEnv = &TestEnv{
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
		ctx:    ctx,
	}

	return testEnv
}

func setupContainers(t *testing.T, ctx context.Context) *TestEnv {
	logger, _ := zap.NewDevelopment()

	// Start Postgres container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("sigec_test"),
		postgres.WithUsername("sigec"),
		postgres.WithPassword("sigec_test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	// Get Postgres connection string
	pgHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get postgres host: %v", err)
	}

	pgPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get postgres port: %v", err)
	}

	pgConnStr := fmt.Sprintf("postgres://sigec:sigec_test@%s:%s/sigec_test?sslmode=disable", pgHost, pgPort.Port())

	// Connect to Postgres
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}

	// Wait for connection
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	// Start Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start redis container: %v", err)
	}

	// Get Redis connection string
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get redis host: %v", err)
	}

	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("Failed to get redis port: %v", err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}

	testEnv = &TestEnv{
		DB:                db,
		Redis:             redisClient,
		PostgresContainer: postgresContainer,
		RedisContainer:    redisContainer,
		Logger:            logger,
		ctx:               ctx,
	}

	return testEnv
}

// TeardownTestEnvironment cleans up the test environment
func TeardownTestEnvironment(t *testing.T) {
	if testEnv == nil {
		return
	}

	ctx := context.Background()

	if testEnv.DB != nil {
		testEnv.DB.Close()
	}

	if testEnv.Redis != nil {
		testEnv.Redis.Close()
	}

	if testEnv.PostgresContainer != nil {
		if err := testEnv.PostgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate postgres container: %v", err)
		}
	}

	if testEnv.RedisContainer != nil {
		if err := testEnv.RedisContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate redis container: %v", err)
		}
	}

	testEnv = nil
}

// CleanDatabase truncates all tables
func CleanDatabase(t *testing.T, db *sql.DB) {
	tables := []string{
		"wallet_transactions",
		"wallets",
		"refunds",
		"payments",
		"payment_cards",
		"transactions",
		"connectors",
		"charge_points",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Table might not exist, that's ok
			t.Logf("Failed to truncate %s: %v", table, err)
		}
	}
}

// FlushRedis clears all Redis keys
func FlushRedis(t *testing.T, client *redis.Client) {
	ctx := context.Background()
	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush redis: %v", err)
	}
}

// SetupSchema creates the database schema for testing
func SetupSchema(t *testing.T, db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		status VARCHAR(50) DEFAULT 'Active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS charge_points (
		id VARCHAR(36) PRIMARY KEY,
		vendor VARCHAR(255),
		model VARCHAR(255),
		serial_number VARCHAR(255),
		firmware_version VARCHAR(100),
		status VARCHAR(50) DEFAULT 'Available',
		latitude DECIMAL(10, 8),
		longitude DECIMAL(11, 8),
		address TEXT,
		last_heartbeat TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS connectors (
		id VARCHAR(36) PRIMARY KEY,
		charge_point_id VARCHAR(36) REFERENCES charge_points(id),
		connector_id INTEGER NOT NULL,
		type VARCHAR(50),
		max_power DECIMAL(10, 2),
		status VARCHAR(50) DEFAULT 'Available',
		UNIQUE(charge_point_id, connector_id)
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id VARCHAR(36) PRIMARY KEY,
		charge_point_id VARCHAR(36) REFERENCES charge_points(id),
		connector_id INTEGER NOT NULL,
		user_id VARCHAR(36) REFERENCES users(id),
		id_tag VARCHAR(100),
		status VARCHAR(50) DEFAULT 'Active',
		meter_start DECIMAL(15, 4) DEFAULT 0,
		meter_stop DECIMAL(15, 4) DEFAULT 0,
		start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		end_time TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS payments (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) REFERENCES users(id),
		transaction_id VARCHAR(36),
		provider VARCHAR(50),
		provider_id VARCHAR(255),
		method VARCHAR(50),
		status VARCHAR(50) DEFAULT 'pending',
		amount DECIMAL(15, 2),
		currency VARCHAR(10) DEFAULT 'BRL',
		description TEXT,
		failure_reason TEXT,
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS wallets (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) UNIQUE REFERENCES users(id),
		balance DECIMAL(15, 2) DEFAULT 0,
		currency VARCHAR(10) DEFAULT 'BRL',
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS wallet_transactions (
		id VARCHAR(36) PRIMARY KEY,
		wallet_id VARCHAR(36) REFERENCES wallets(id),
		user_id VARCHAR(36) REFERENCES users(id),
		type VARCHAR(20),
		amount DECIMAL(15, 2),
		balance DECIMAL(15, 2),
		description TEXT,
		reference_id VARCHAR(36),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_charge_point_id ON transactions(charge_point_id);
	CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
}
