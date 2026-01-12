# github-copilot-agent-demo
A repository used to demo Github Copilot Agent

# Goal
The goal of this project is to demonstrate how Github Copilot Agent can be used to implement entire projects/features, instead of simply helping with auto-complete suggestions.

This project will consist of a Go application that Connects to a Cosmos DB account on Azure, using a User-Assigned Managed Service Identity to authenticate.

# Requirements
- Go application
- uses the AzIdentity and AzTables SDK
- Uses the `azidentity.NewManagedIdentityCredential` API to create a token credential
- Generates Promethus logs on connection error

# Architecture
- Single Go application that runs on AKS

# Implementation
The application has been implemented with the following features:
- Connects to Azure Cosmos DB Table API using the AzTables SDK
- Authenticates using Managed Service Identity (both User-Assigned and System-Assigned)
- Exposes Prometheus metrics at `/metrics` endpoint
- Tracks connection errors via `cosmos_connection_errors_total` counter
- Configurable via environment variables

# Usage

## Building
```bash
go build
```

## Running
The application requires the following environment variables:

- `COSMOS_ACCOUNT_NAME` (required): Name of your Cosmos DB account
- `MANAGED_IDENTITY_CLIENT_ID` (optional): Client ID for User-Assigned Managed Identity. If not set, System-Assigned Managed Identity is used
- `TABLE_NAME` (optional): Name of the table to connect to. Defaults to "DefaultTable"
- `METRICS_PORT` (optional): Port for Prometheus metrics endpoint. Defaults to "8080"

Example:
```bash
export COSMOS_ACCOUNT_NAME=mycosmosaccount
export MANAGED_IDENTITY_CLIENT_ID=12345678-1234-1234-1234-123456789abc
export TABLE_NAME=mytable
export METRICS_PORT=8080
./github-copilot-agent-demo
```

## Metrics
The application exposes Prometheus metrics at `http://localhost:8080/metrics` (or the configured METRICS_PORT).

Available metrics:
- `cosmos_connection_errors_total`: Total number of connection errors to Cosmos DB

## Dependencies
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity` - Azure Identity SDK for Managed Identity authentication
- `github.com/Azure/azure-sdk-for-go/sdk/data/aztables` - Azure Tables SDK for Cosmos DB operations
- `github.com/prometheus/client_golang` - Prometheus client library for metrics
