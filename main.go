package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Prometheus metric for connection errors
	cosmosConnectionErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cosmos_connection_errors_total",
			Help: "Total number of Cosmos DB connection errors",
		},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(cosmosConnectionErrors)
}

func main() {
	// Get configuration from environment variables
	cosmosAccountName := os.Getenv("COSMOS_ACCOUNT_NAME")
	tableName := os.Getenv("TABLE_NAME")
	managedIdentityClientIDs := parseManagedIdentityClientIDs(os.Getenv("MANAGED_IDENTITY_CLIENT_IDS"))

	if cosmosAccountName == "" {
		log.Fatal("COSMOS_ACCOUNT_NAME environment variable is required")
	}

	if len(managedIdentityClientIDs) == 0 {
		log.Fatal("MANAGED_IDENTITY_CLIENT_IDS environment variable is required")
	}

	if tableName == "" {
		tableName = "DefaultTable"
		log.Printf("TABLE_NAME not set, using default: %s", tableName)
	}

	serviceURL := fmt.Sprintf("https://%s.table.cosmos.azure.com", cosmosAccountName)
	if err := testCosmosConnections(context.Background(), serviceURL, cosmosAccountName, tableName, managedIdentityClientIDs); err != nil {
		log.Fatal(err)
	}

	// Start Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8080"
	}

	log.Printf("Starting metrics server on port %s", metricsPort)
	if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}

func parseManagedIdentityClientIDs(value string) []string {
	parts := strings.Split(value, ",")
	clientIDs := make([]string, 0, len(parts))
	for _, part := range parts {
		clientID := strings.TrimSpace(part)
		if clientID != "" {
			clientIDs = append(clientIDs, clientID)
		}
	}
	return clientIDs
}

func testCosmosConnections(ctx context.Context, serviceURL, cosmosAccountName, tableName string, managedIdentityClientIDs []string) error {
	errCh := make(chan error, len(managedIdentityClientIDs))
	var wg sync.WaitGroup

	for i, clientID := range managedIdentityClientIDs {
		wg.Add(1)
		go func(index int, clientID string) {
			defer wg.Done()

			cred, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
				ID: azidentity.ClientID(clientID),
			})
			if err != nil {
				errCh <- fmt.Errorf("managed identity %d: failed to create credential: %w", index+1, err)
				return
			}

			serviceClient, err := aztables.NewServiceClient(serviceURL, cred, nil)
			if err != nil {
				errCh <- fmt.Errorf("managed identity %d: failed to create Cosmos DB service client: %w", index+1, err)
				return
			}

			if _, err := serviceClient.GetProperties(ctx, nil); err != nil {
				errCh <- fmt.Errorf("managed identity %d: failed to connect to Cosmos DB: %w", index+1, err)
				return
			}

			log.Printf("Successfully connected to Cosmos DB with managed identity %d/%d (account: %s, table: %s)", index+1, len(managedIdentityClientIDs), cosmosAccountName, tableName)
		}(i, clientID)
	}

	wg.Wait()
	close(errCh)

	var failed bool
	for err := range errCh {
		failed = true
		log.Print(err)
		cosmosConnectionErrors.Inc()
	}

	if failed {
		return fmt.Errorf("one or more Cosmos DB connections failed")
	}

	log.Printf("Successfully connected to Cosmos DB with %d managed identities (account: %s, table: %s)", len(managedIdentityClientIDs), cosmosAccountName, tableName)
	return nil
}
