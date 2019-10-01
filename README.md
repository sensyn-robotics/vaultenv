# vaultenv [![Build Status](https://dev.azure.com/sensyn-robotics/vaultenv/_apis/build/status/sensyn-robotics.vaultenv?branchName=master)](https://dev.azure.com/sensyn-robotics/vaultenv/_build/latest?definitionId=1&branchName=master) ![Azure DevOps coverage (branch)](https://img.shields.io/azure-devops/coverage/sensyn-robotics/vaultenv/1/master)
Replace Azure Keyvault
## Installation
```
go get github.com/sensyn-robotics/vaultenv
```
## Usage
### Permit get secrets on your VM
```
$ az vm identity assign --name <NameOfYourVirtualMachine> --resource-group <YourResourceGroupName>
{
  "systemAssignedIdentity": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "userAssignedIdentities": {}
}
$ az keyvault set-policy --name <YourKeyVaultName> --object-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --secret-permissions get
```
see detail https://docs.microsoft.com/azure/key-vault/tutorial-net-linux-virtual-machine#assign-an-identity-to-the-vm
### Filter .env
```
$ cat .env
USER1=user1
PASSWORD1={{ kv "https://keyvault-name.vault.azure.net/secrets/example-password" }}
$ cat .env | vaultenv
USER1=user1
PASSWORD1=SecretsFromAzureKeyVault
```
