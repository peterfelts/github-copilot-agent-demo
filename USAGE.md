# Usage Guide

## Overview
This Go application connects to Azure Cosmos DB (Table API) using a User-Assigned Managed Service Identity for authentication. It exposes Prometheus metrics for monitoring connection errors and provides health check endpoints.

## Prerequisites
- Go 1.24 or later
- Azure subscription with:
  - Azure Cosmos DB account (with Table API enabled)
  - User-Assigned Managed Identity
  - AKS cluster (for deployment)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `COSMOS_ACCOUNT_NAME` | Yes | - | Name of your Cosmos DB account |
| `COSMOS_TABLE_NAME` | No | `defaultTable` | Name of the table to connect to |
| `MANAGED_IDENTITY_CLIENT_ID` | No | - | Client ID of the user-assigned managed identity. If not provided, uses system-assigned identity |
| `METRICS_PORT` | No | `8080` | Port for the metrics and health endpoints |

## Building the Application

### Local Build
```bash
go build -o app main.go
```

### Docker Build
```bash
docker build -t cosmos-app:latest .
```

## Running Locally

```bash
export COSMOS_ACCOUNT_NAME="your-cosmos-account"
export COSMOS_TABLE_NAME="your-table-name"
export MANAGED_IDENTITY_CLIENT_ID="your-managed-identity-client-id"
./app
```

## Deploying to AKS

### 1. Set up Azure Resources

#### Create User-Assigned Managed Identity
```bash
az identity create --name cosmos-app-identity --resource-group <your-rg>
```

#### Get the Managed Identity details
```bash
az identity show --name cosmos-app-identity --resource-group <your-rg>
```

#### Assign permissions to Cosmos DB
```bash
# Get the identity's principal ID
PRINCIPAL_ID=$(az identity show --name cosmos-app-identity --resource-group <your-rg> --query principalId -o tsv)

# Assign the appropriate Cosmos DB role
az cosmosdb sql role assignment create \
  --account-name <your-cosmos-account> \
  --resource-group <your-rg> \
  --scope "/" \
  --principal-id $PRINCIPAL_ID \
  --role-definition-name "Cosmos DB Built-in Data Contributor"
```

### 2. Build and Push Docker Image

```bash
# Log in to Azure Container Registry
az acr login --name <your-acr>

# Build and tag the image
docker build -t <your-acr>.azurecr.io/cosmos-app:latest .

# Push to ACR
docker push <your-acr>.azurecr.io/cosmos-app:latest
```

### 3. Configure AKS to use Managed Identity

```bash
# Enable pod identity on AKS cluster
az aks update \
  --resource-group <your-rg> \
  --name <your-aks-cluster> \
  --enable-managed-identity \
  --enable-pod-identity

# Create pod identity
az aks pod-identity add \
  --resource-group <your-rg> \
  --cluster-name <your-aks-cluster> \
  --namespace default \
  --name cosmos-app-identity \
  --identity-resource-id "/subscriptions/<subscription-id>/resourcegroups/<your-rg>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cosmos-app-identity"
```

### 4. Deploy to Kubernetes

Update the `k8s-deployment.yaml` file with your specific values:
- Container image URL
- Cosmos DB account name
- Table name
- Managed Identity Client ID

Then apply the deployment:

```bash
kubectl apply -f k8s-deployment.yaml
```

### 5. Verify Deployment

```bash
# Check pod status
kubectl get pods -l app=cosmos-app

# Check logs
kubectl logs -l app=cosmos-app

# Port forward to access metrics locally
kubectl port-forward svc/cosmos-app-service 8080:8080
```

## Endpoints

- **Health Check**: `http://localhost:8080/health`
  - Returns 200 OK if the application is running
  
- **Metrics**: `http://localhost:8080/metrics`
  - Prometheus-formatted metrics including `cosmos_connection_errors_total`

## Prometheus Metrics

The application exposes the following metrics:

- `cosmos_connection_errors_total{error_type="<type>"}`: Counter for connection errors
  - `error_type="credential_creation"`: Failed to create managed identity credential
  - `error_type="service_client_creation"`: Failed to create Azure Tables service client
  - `error_type="service_connection"`: Failed to connect to Cosmos DB service

## Monitoring Setup

To scrape metrics with Prometheus in Kubernetes, add the following annotations to the deployment:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

## Troubleshooting

### Common Issues

1. **"COSMOS_ACCOUNT_NAME environment variable is required"**
   - Ensure the environment variable is set in your deployment

2. **"Failed to create managed identity credential"**
   - Verify the managed identity is properly configured
   - Check if the pod identity is correctly set up in AKS

3. **"Failed to connect to Cosmos DB service"**
   - Verify the Cosmos DB account name is correct
   - Ensure the managed identity has the appropriate permissions
   - Check network connectivity from AKS to Cosmos DB

4. **Connection errors in logs**
   - Check Prometheus metrics at `/metrics` endpoint for detailed error counts
   - Review IAM permissions on the Cosmos DB account
   - Verify the managed identity client ID is correct

## Security Best Practices

- Use User-Assigned Managed Identity for better control and separation of concerns
- Follow the principle of least privilege when assigning Cosmos DB permissions
- Regularly rotate and audit managed identity permissions
- Use network policies to restrict access to the application
- Enable Azure Policy to enforce security standards

## Additional Resources

- [Azure SDK for Go Documentation](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go)
- [Azure Managed Identities](https://docs.microsoft.com/azure/active-directory/managed-identities-azure-resources/)
- [Azure Cosmos DB Table API](https://docs.microsoft.com/azure/cosmos-db/table-introduction)
- [Prometheus Metrics](https://prometheus.io/docs/concepts/metric_types/)
