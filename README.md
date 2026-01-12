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
