package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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
	managedIdentityClientID := os.Getenv("MANAGED_IDENTITY_CLIENT_ID")

	if cosmosAccountName == "" {
		log.Fatal("COSMOS_ACCOUNT_NAME environment variable is required")
	}

	if tableName == "" {
		tableName = "DefaultTable"
		log.Printf("TABLE_NAME not set, using default: %s", tableName)
	}

	// Create Managed Identity credential
	var cred *azidentity.ManagedIdentityCredential
	var err error

	if managedIdentityClientID != "" {
		// Use User-Assigned Managed Identity
		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(managedIdentityClientID),
		})
	} else {
		// Use System-Assigned Managed Identity
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	}

	if err != nil {
		log.Printf("Failed to create managed identity credential: %v", err)
		cosmosConnectionErrors.Inc()
		log.Fatal(err)
	}

	// Create Cosmos DB table service client
	serviceURL := fmt.Sprintf("https://%s.table.cosmos.azure.com", cosmosAccountName)
	serviceClient, err := aztables.NewServiceClient(serviceURL, cred, nil)
	if err != nil {
		log.Printf("Failed to create Cosmos DB service client: %v", err)
		cosmosConnectionErrors.Inc()
		log.Fatal(err)
	}

	// Test connection by attempting to get service properties
	ctx := context.Background()
	_, err = serviceClient.GetProperties(ctx, nil)
	if err != nil {
		log.Printf("Failed to connect to Cosmos DB: %v", err)
		cosmosConnectionErrors.Inc()
		log.Fatal(err)
	}

	log.Printf("Successfully connected to Cosmos DB (account: %s, table: %s)", cosmosAccountName, tableName)

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
