package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	goNietzscheDB "github.com/NietzscheDB/go-NietzscheDB/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/NietzscheDB"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// TestEnv holds test environment resources
type TestEnv struct {
	NietzscheDB          *goNietzscheDB.Client
	NietzscheDBContainer testcontainers.Container
	Logger         *zap.Logger
	ctx            context.Context
	// NietzscheDB connection is handled via nietzsche-sdk in each test
	NietzscheAddr  string // gRPC address for NietzscheDB
	NietzscheColl  string // NietzscheDB collection name
}

var testEnv *TestEnv

// SetupTestEnvironment initializes the test environment with containers
func SetupTestEnvironment(t *testing.T) *TestEnv {
	if testEnv != nil {
		return testEnv
	}

	ctx := context.Background()

	// Use testcontainers for local NietzscheDB testing
	return setupContainers(t, ctx)
}

func setupContainers(t *testing.T, ctx context.Context) *TestEnv {
	logger, _ := zap.NewDevelopment()

	// Start NietzscheDB container
	NietzscheDBContainer, err := NietzscheDB.RunContainer(ctx,
		testcontainers.WithImage("NietzscheDB:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start NietzscheDB container: %v", err)
	}

	// Get NietzscheDB connection string
	NietzscheDBHost, err := NietzscheDBContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get NietzscheDB host: %v", err)
	}

	NietzscheDBPort, err := NietzscheDBContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("Failed to get NietzscheDB port: %v", err)
	}

	// Connect to NietzscheDB
	NietzscheDBClient := goNietzscheDB.NewClient(&goNietzscheDB.Options{
		Addr: fmt.Sprintf("%s:%s", NietzscheDBHost, NietzscheDBPort.Port()),
	})

	if err := NietzscheDBClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to NietzscheDB: %v", err)
	}

	testEnv = &TestEnv{
		NietzscheDB:          NietzscheDBClient,
		NietzscheDBContainer: NietzscheDBContainer,
		Logger:         logger,
		ctx:            ctx,
		NietzscheAddr:  "136.111.0.47:50051",
		NietzscheColl:  "ev-ia-test",
	}

	return testEnv
}

// TeardownTestEnvironment cleans up the test environment
func TeardownTestEnvironment(t *testing.T) {
	if testEnv == nil {
		return
	}

	ctx := context.Background()

	if testEnv.NietzscheDB != nil {
		testEnv.NietzscheDB.Close()
	}

	if testEnv.NietzscheDBContainer != nil {
		if err := testEnv.NietzscheDBContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate NietzscheDB container: %v", err)
		}
	}

	testEnv = nil
}

// FlushNietzscheDB clears all NietzscheDB keys
func FlushNietzscheDB(t *testing.T, client *goNietzscheDB.Client) {
	ctx := context.Background()
	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush NietzscheDB: %v", err)
	}
}
