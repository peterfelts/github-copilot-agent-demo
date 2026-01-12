package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cosmosConnectionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cosmos_connection_errors_total",
		Help: "The total number of Cosmos DB connection errors",
	})
)

func main() {
	ctx := context.Background()

	// Get configuration from environment variables
	cosmosAccountName := os.Getenv("COSMOS_ACCOUNT_NAME")
	tableName := os.Getenv("TABLE_NAME")
	clientID := os.Getenv("MANAGED_IDENTITY_CLIENT_ID")

	if cosmosAccountName == "" {
		log.Fatal("COSMOS_ACCOUNT_NAME environment variable is required")
	}

	if tableName == "" {
		log.Fatal("TABLE_NAME environment variable is required")
	}

	// Create managed identity credential
	var cred *azidentity.ManagedIdentityCredential
	var err error

	if clientID != "" {
		// Use user-assigned managed identity with specific client ID
		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(clientID),
		})
	} else {
		// Use system-assigned managed identity
		cred, err = azidentity.NewManagedIdentityCredential(nil)
	}

	if err != nil {
		cosmosConnectionErrors.Inc()
		log.Fatalf("Failed to create managed identity credential: %v", err)
	}

	// Create the service URL for Cosmos DB Table API
	serviceURL := fmt.Sprintf("https://%s.table.cosmos.azure.com", cosmosAccountName)

	// Create a new Azure Tables service client
	serviceClient, err := aztables.NewServiceClient(serviceURL, cred, nil)
	if err != nil {
		cosmosConnectionErrors.Inc()
		log.Fatalf("Failed to create service client: %v", err)
	}

	// Create a table client
	tableClient := serviceClient.NewClient(tableName)

	// Test the connection by attempting to create the table if it doesn't exist
	_, err = tableClient.CreateTable(ctx, nil)
	if err != nil {
		// Check if it's a "table already exists" error, which is acceptable
		errMsg := err.Error()
		if !strings.Contains(strings.ToLower(errMsg), "already exists") &&
			!strings.Contains(strings.ToLower(errMsg), "tablebeingdeleted") {
			// If it's not a "table exists" error, increment the metric and fail
			cosmosConnectionErrors.Inc()
			log.Fatalf("Failed to connect to Cosmos DB table: %v", err)
		}
		log.Printf("Table operation info: %v", err)
	}

	log.Printf("Successfully connected to Cosmos DB account: %s, table: %s", cosmosAccountName, tableName)
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Run the application continuously
	log.Println("Application running. Press Ctrl+C to stop...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Periodic health check
			log.Println("Health check: Application is running")
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			return
		}
	}
}
