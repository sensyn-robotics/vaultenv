[![Build Status](https://github.com/sensyn-robotics/vaultenv/workflows/ci/badge.svg?branch=master)](https://github.com/sensyn-robotics/vaultenv/actions?query=branch%3Amaster) [![Codecov](https://img.shields.io/codecov/c/github/sensyn-robotics/vaultenv/master)](https://codecov.io/gh/sensyn-robotics/vaultenv/branch/master)
# vaultenv 
Replace Azure Keyvault Secret Identifier written into .env etc.
## Installation
### Prebuilt binary
Download the linux/amd64 archive from the [Releases](https://github.com/sensyn-robotics/vaultenv/releases) page and extract it.
```
tar xzf vaultenv_<version>_linux_amd64.tar.gz
./vaultenv -version
```
### go install
```
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
## Release
Push a `v*` tag. The release workflow runs [GoReleaser](https://goreleaser.com/), which creates a GitHub Release and attaches the linux/amd64 archive and `checksums.txt`.
```
git tag v1.2.3
git push origin v1.2.3
```
