[![Build Status](https://github.com/sensyn-robotics/vaultenv/workflows/ci/badge.svg?branch=master)](https://github.com/sensyn-robotics/vaultenv/actions?query=branch%3Amaster) [![Codecov](https://img.shields.io/codecov/c/github/sensyn-robotics/vaultenv/master)](https://codecov.io/gh/sensyn-robotics/vaultenv/branch/master)
# vaultenv 
Replace Azure Keyvault Secret Identifier written into .env etc.
## Installation
```
go get github.com/sensyn-robotics/vaultenv
# for go v1.18 and above
go install github.com/sensyn-robotics/vaultenv@latest
```
## Usage
###
* Use service princilpal
```
$ export VAULTENV_AZURE_USER=<service principal id>
$ export VAULTENV_AZURE_PASSWORD=<service principal secret>
$ export VAULTENV_AZURE_TENANT=<tenant id>
```
see detail https://docs.microsoft.com/en-us/azure/key-vault/general/group-permissions-for-apps#applications

* or Use VM Identity
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
