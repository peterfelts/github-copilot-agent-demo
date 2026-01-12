# Infrastructure Deployment

This directory contains Bicep templates for deploying the Go application to Azure VM with Cosmos DB.

## Architecture

The infrastructure includes:

- **Azure Virtual Machine**: Ubuntu 22.04 LTS running the Go application
- **Cosmos DB**: Table API enabled for data storage
- **User-Assigned Managed Identity**: For secure authentication to Cosmos DB
- **Virtual Network**: Isolated network with subnet
- **Network Security Group**: Allows SSH (22) and metrics endpoint (8080)
- **Public IP**: For external access to the VM

## Prerequisites

1. Azure subscription
2. Azure CLI installed
3. Appropriate permissions to create resources
4. SSH public key for VM access

## Manual Deployment

### 1. Login to Azure

```bash
az login
az account set --subscription <your-subscription-id>
```

### 2. Create Resource Group

```bash
az group create --name copilot-demo-rg --location eastus
```

### 3. Deploy Infrastructure

```bash
# Generate unique Cosmos DB name
UNIQUE_SUFFIX=$(date +%s | sha256sum | cut -c1-8)
COSMOS_NAME="copilot-demo-${UNIQUE_SUFFIX}"

# Deploy with your SSH public key
az deployment group create \
  --resource-group copilot-demo-rg \
  --template-file main.bicep \
  --parameters cosmosAccountName="${COSMOS_NAME}" \
  --parameters sshPublicKey="$(cat ~/.ssh/id_rsa.pub)"
```

### 4. Get Deployment Outputs

```bash
az deployment group show \
  --resource-group copilot-demo-rg \
  --name main \
  --query properties.outputs
```

## Automated Deployment via GitHub Actions

The repository includes a GitHub Actions workflow that:

1. Builds the Go application
2. Deploys Azure infrastructure using Bicep
3. Validates the deployment
4. Tests connectivity to Cosmos DB

### Required GitHub Secrets

Configure the following secrets in your GitHub repository:

- `AZURE_CLIENT_ID`: Service principal client ID
- `AZURE_TENANT_ID`: Azure tenant ID
- `AZURE_SUBSCRIPTION_ID`: Azure subscription ID
- `SSH_PUBLIC_KEY` (optional): SSH public key for VM access

### Setting up Azure Service Principal

```bash
# Create service principal with contributor role
az ad sp create-for-rbac \
  --name "github-copilot-demo-sp" \
  --role contributor \
  --scopes /subscriptions/<your-subscription-id> \
  --sdk-auth
```

Use the output to configure GitHub secrets.

### Running the Workflow

The workflow runs automatically on:
- Push to `main` branch
- Pull requests to `main` branch
- Manual trigger via GitHub Actions UI

To manually trigger:
1. Go to Actions tab in GitHub
2. Select "Deploy to Azure VM and Validate"
3. Click "Run workflow"

## Validation

The deployment validation includes:

1. **VM Status Check**: Ensures VM is running
2. **Metrics Endpoint Test**: Verifies the application is accessible at `http://<VM-IP>:8080/metrics`
3. **Cosmos DB Connection**: Checks that the application successfully connects to Cosmos DB

## Accessing the Deployed Application

After deployment, you can:

1. **View Metrics**: 
   ```bash
   curl http://<VM-PUBLIC-IP>:8080/metrics
   ```

2. **SSH to VM**:
   ```bash
   ssh azureuser@<VM-PUBLIC-IP>
   ```

3. **Check Application Logs**:
   ```bash
   ssh azureuser@<VM-PUBLIC-IP>
   sudo journalctl -u copilot-demo.service -f
   ```

## Configuration

### Bicep Parameters

Customize deployment by modifying parameters in `main.bicep`:

- `location`: Azure region (default: eastus)
- `prefix`: Prefix for resource names
- `cosmosAccountName`: Cosmos DB account name (must be globally unique)
- `tableName`: Name of the Cosmos DB table
- `vmSize`: VM size (default: Standard_B2s)
- `adminUsername`: Admin username for VM
- `sshPublicKey`: SSH public key for authentication (format: ssh-rsa AAAAB3... user@host)
- `repositoryUrl`: GitHub repository URL to clone (default: this repository)
- `goVersion`: Go version to install (default: 1.21.5)

### Application Configuration

The application is configured via environment variables set in `/etc/environment`:

- `MANAGED_IDENTITY_CLIENT_ID`: Automatically set from deployed identity
- `COSMOS_ACCOUNT_NAME`: Automatically set from deployed Cosmos DB
- `TABLE_NAME`: Automatically set from parameters
- `METRICS_PORT`: Set to 8080

## Cleanup

To remove all deployed resources:

```bash
az group delete --name copilot-demo-rg --yes --no-wait
```

## Troubleshooting

### VM not accessible
- Check NSG rules allow port 8080
- Verify public IP is correctly assigned
- Check VM status: `az vm get-instance-view --resource-group copilot-demo-rg --name copilot-demo-vm`

### Application not running
SSH to VM and check:
```bash
sudo systemctl status copilot-demo.service
sudo journalctl -u copilot-demo.service -n 50
```

### Cosmos DB connection issues
- Verify managed identity has correct permissions
- Check role assignments on Cosmos DB account
- Review application logs for authentication errors

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Azure Subscription                    │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │         Resource Group: copilot-demo-rg            │ │
│  │                                                    │ │
│  │  ┌──────────────┐      ┌─────────────────────┐   │ │
│  │  │   Virtual    │      │  User-Assigned      │   │ │
│  │  │   Machine    │─────▶│  Managed Identity   │   │ │
│  │  │  (Ubuntu)    │      └─────────────────────┘   │ │
│  │  │              │               │                 │ │
│  │  │ Go App:8080  │               │ Authenticates   │ │
│  │  └──────────────┘               │                 │ │
│  │         │                        ▼                 │ │
│  │         │              ┌─────────────────────┐    │ │
│  │         │              │    Cosmos DB        │    │ │
│  │         │              │   (Table API)       │    │ │
│  │         │              └─────────────────────┘    │ │
│  │         │                                          │ │
│  │         │ Public IP                                │ │
│  │         ▼                                          │ │
│  │  ┌──────────────┐                                 │ │
│  │  │   Internet   │◀─── Metrics: /metrics           │ │
│  │  │    Users     │◀─── SSH Access                  │ │
│  │  └──────────────┘                                 │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```
