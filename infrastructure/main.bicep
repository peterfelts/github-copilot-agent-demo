@description('Azure region for resources')
param location string = 'eastus'

@description('GitHub repository URL for the application')
param repositoryUrl string = 'https://github.com/peterfelts/github-copilot-agent-demo.git'

@description('Go version to install')
param goVersion string = '1.21.5'

@description('Prefix for resource names')
param prefix string = 'copilot-demo'

@description('Name of the Cosmos DB account (must be globally unique)')
param cosmosAccountName string

@description('Name of the Cosmos DB table')
param tableName string = 'DemoTable'

@description('Size of the virtual machine')
param vmSize string = 'Standard_B2s'

@description('Admin username for the VM')
param adminUsername string = 'azureuser'

@description('SSH public key for VM access')
@secure()
param sshPublicKey string

@description('Tags to apply to all resources')
param tags object = {
  Environment: 'Demo'
  Project: 'GitHub-Copilot-Agent-Demo'
}

// Virtual Network
resource vnet 'Microsoft.Network/virtualNetworks@2023-05-01' = {
  name: '${prefix}-vnet'
  location: location
  tags: tags
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
    subnets: [
      {
        name: '${prefix}-subnet'
        properties: {
          addressPrefix: '10.0.1.0/24'
          networkSecurityGroup: {
            id: nsg.id
          }
        }
      }
    ]
  }
}

// Network Security Group
resource nsg 'Microsoft.Network/networkSecurityGroups@2023-05-01' = {
  name: '${prefix}-nsg'
  location: location
  tags: tags
  properties: {
    securityRules: [
      {
        name: 'AllowMetrics'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '8080'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
      {
        name: 'AllowSSH'
        properties: {
          priority: 101
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '22'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
    ]
  }
}

// Public IP
resource publicIP 'Microsoft.Network/publicIPAddresses@2023-05-01' = {
  name: '${prefix}-pip'
  location: location
  tags: tags
  sku: {
    name: 'Standard'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
  }
}

// Network Interface
resource nic 'Microsoft.Network/networkInterfaces@2023-05-01' = {
  name: '${prefix}-nic'
  location: location
  tags: tags
  properties: {
    ipConfigurations: [
      {
        name: 'internal'
        properties: {
          subnet: {
            id: vnet.properties.subnets[0].id
          }
          privateIPAllocationMethod: 'Dynamic'
          publicIPAddress: {
            id: publicIP.id
          }
        }
      }
    ]
  }
}

// User Assigned Managed Identity
resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${prefix}-identity'
  location: location
  tags: tags
}

// Cosmos DB Account
resource cosmosAccount 'Microsoft.DocumentDB/databaseAccounts@2023-04-15' = {
  name: cosmosAccountName
  location: location
  tags: tags
  kind: 'GlobalDocumentDB'
  properties: {
    databaseAccountOfferType: 'Standard'
    capabilities: [
      {
        name: 'EnableTable'
      }
    ]
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
    locations: [
      {
        locationName: location
        failoverPriority: 0
        isZoneRedundant: false
      }
    ]
  }
}

// Cosmos DB Table
resource cosmosTable 'Microsoft.DocumentDB/databaseAccounts/tables@2023-04-15' = {
  parent: cosmosAccount
  name: tableName
  properties: {
    resource: {
      id: tableName
    }
    options: {
      throughput: 400
    }
  }
}

// Role Assignment - Cosmos DB Account Reader
resource cosmosReaderRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(cosmosAccount.id, identity.id, 'reader')
  scope: cosmosAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'fbdf93bf-df7d-467e-a4d2-9458aa1360c8') // Cosmos DB Account Reader Role
    principalId: identity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

// Role Assignment - Cosmos DB Built-in Data Contributor
// Note: This uses the preview Cosmos DB RBAC role for Table API access
resource cosmosDataContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(cosmosAccount.id, identity.id, 'contributor')
  scope: cosmosAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c') // Contributor role for broader access
    principalId: identity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

// Virtual Machine
resource vm 'Microsoft.Compute/virtualMachines@2023-03-01' = {
  name: '${prefix}-vm'
  location: location
  tags: tags
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identity.id}': {}
    }
  }
  properties: {
    hardwareProfile: {
      vmSize: vmSize
    }
    osProfile: {
      computerName: '${prefix}-vm'
      adminUsername: adminUsername
      customData: base64(loadTextContent('./scripts/cloud-init.yaml'))
      linuxConfiguration: {
        disablePasswordAuthentication: true
        ssh: {
          publicKeys: [
            {
              path: '/home/${adminUsername}/.ssh/authorized_keys'
              keyData: sshPublicKey
            }
          ]
        }
      }
    }
    storageProfile: {
      imageReference: {
        publisher: 'Canonical'
        offer: '0001-com-ubuntu-server-jammy'
        sku: '22_04-lts-gen2'
        version: 'latest'
      }
      osDisk: {
        createOption: 'FromImage'
        managedDisk: {
          storageAccountType: 'Premium_LRS'
        }
      }
    }
    networkProfile: {
      networkInterfaces: [
        {
          id: nic.id
        }
      ]
    }
  }
}

// VM Extension to configure the application
resource vmExtension 'Microsoft.Compute/virtualMachines/extensions@2023-03-01' = {
  parent: vm
  name: 'configureApp'
  location: location
  properties: {
    publisher: 'Microsoft.Azure.Extensions'
    type: 'CustomScript'
    typeHandlerVersion: '2.1'
    autoUpgradeMinorVersion: true
    settings: {
      commandToExecute: 'echo "MANAGED_IDENTITY_CLIENT_ID=${identity.properties.clientId}" >> /etc/environment && echo "COSMOS_ACCOUNT_NAME=${cosmosAccount.name}" >> /etc/environment && echo "TABLE_NAME=${tableName}" >> /etc/environment && echo "METRICS_PORT=8080" >> /etc/environment && echo "REPOSITORY_URL=${repositoryUrl}" >> /etc/environment && echo "GO_VERSION=${goVersion}" >> /etc/environment'
    }
  }
}

// Outputs
output vmPublicIP string = publicIP.properties.ipAddress
output vmName string = vm.name
output cosmosAccountName string = cosmosAccount.name
output cosmosEndpoint string = cosmosAccount.properties.documentEndpoint
output managedIdentityClientId string = identity.properties.clientId
output tableName string = tableName
output metricsEndpoint string = 'http://${publicIP.properties.ipAddress}:8080/metrics'
