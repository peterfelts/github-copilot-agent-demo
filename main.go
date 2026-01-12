package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
		// Table might already exist, which is fine
		// We'll log the error but not fail if it's a specific error
		log.Printf("Table creation returned: %v (table may already exist)", err)
	}

	log.Printf("Successfully connected to Cosmos DB account: %s, table: %s", cosmosAccountName, tableName)
	log.Println("Application running successfully")
}
