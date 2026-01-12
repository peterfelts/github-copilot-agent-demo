package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Prometheus metric for connection errors
	cosmosConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cosmos_connection_errors_total",
			Help: "Total number of Cosmos DB connection errors",
		},
		[]string{"error_type"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(cosmosConnectionErrors)
}

func main() {
	log.Println("Starting Cosmos DB connection application...")

	// Get configuration from environment variables
	accountName := os.Getenv("COSMOS_ACCOUNT_NAME")
	tableName := os.Getenv("COSMOS_TABLE_NAME")
	clientID := os.Getenv("MANAGED_IDENTITY_CLIENT_ID")

	if accountName == "" {
		log.Fatal("COSMOS_ACCOUNT_NAME environment variable is required")
	}

	if tableName == "" {
		log.Println("COSMOS_TABLE_NAME not set, using default: 'defaultTable'")
		tableName = "defaultTable"
	}

	// Create managed identity credential
	var cred *azidentity.ManagedIdentityCredential
	var err error

	if clientID != "" {
		// Use user-assigned managed identity with specific client ID
		log.Printf("Using user-assigned managed identity with client ID: %s\n", clientID)
		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(clientID),
		})
	} else {
		// Use system-assigned managed identity
		log.Println("Using system-assigned managed identity")
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	}

	if err != nil {
		cosmosConnectionErrors.WithLabelValues("credential_creation").Inc()
		log.Fatalf("Failed to create managed identity credential: %v", err)
	}

	// Construct the service URL for Cosmos DB Table API
	serviceURL := fmt.Sprintf("https://%s.table.cosmos.azure.com", accountName)

	// Create Azure Tables service client
	serviceClient, err := aztables.NewServiceClient(serviceURL, cred, nil)
	if err != nil {
		cosmosConnectionErrors.WithLabelValues("service_client_creation").Inc()
		log.Fatalf("Failed to create Azure Tables service client: %v", err)
	}

	log.Printf("Successfully created service client for: %s\n", serviceURL)

	// Try to verify connection by getting service properties
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to get service properties to verify connection
	_, err = serviceClient.GetProperties(ctx, nil)
	if err != nil {
		cosmosConnectionErrors.WithLabelValues("service_connection").Inc()
		log.Printf("Warning: Failed to connect to Cosmos DB service: %v", err)
		log.Println("This may be expected if running outside Azure or if permissions are not configured")
	} else {
		log.Printf("Successfully connected to Cosmos DB service\n")
		
		// Verify table access
		tableClient := serviceClient.NewClient(tableName)
		_, err = tableClient.CreateTable(ctx, nil)
		if err != nil {
			// Table might already exist or we might not have permissions - this is okay
			log.Printf("Table '%s' status check: %v (table may already exist)\n", tableName, err)
		} else {
			log.Printf("Successfully verified table: %s\n", tableName)
		}
	}

	// Start Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8080"
	}

	log.Printf("Starting metrics server on port %s\n", metricsPort)
	log.Printf("Metrics available at http://localhost:%s/metrics\n", metricsPort)
	log.Printf("Health check available at http://localhost:%s/health\n", metricsPort)

	if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}
