package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"text/template"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type secretClient interface {
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
}

type clientFactory func(vaultURL string) (secretClient, error)

type fetcher struct {
	factory clientFactory
	clients map[string]secretClient
}

// version is injected at release build time via ldflags (-X main.version=...).
// It is empty for `go install` and local builds, where versionString() falls
// back to the module version embedded by the Go toolchain.
var version = ""

func versionString() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "unknown"
}

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(versionString())
		return
	}

	f, err := newFetcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := filter(f, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newFetcher() (*fetcher, error) {
	cred, err := newCredential()
	if err != nil {
		return nil, err
	}
	factory := func(vaultURL string) (secretClient, error) {
		return azsecrets.NewClient(vaultURL, cred, nil)
	}
	return &fetcher{
		factory: factory,
		clients: make(map[string]secretClient),
	}, nil
}

func newCredential() (azcore.TokenCredential, error) {
	var creds []azcore.TokenCredential

	// 1. Service Principal (from environment variables)
	clientID := os.Getenv("VAULTENV_AZURE_USER")
	clientSecret := os.Getenv("VAULTENV_AZURE_PASSWORD")
	tenantID := os.Getenv("VAULTENV_AZURE_TENANT")

	if clientID != "" && clientSecret != "" && tenantID != "" {
		spCred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("service principal credential: %w", err)
		}
		creds = append(creds, spCred)
	}

	// 2. Azure CLI
	cliCred, err := azidentity.NewAzureCLICredential(nil)
	if err == nil {
		creds = append(creds, cliCred)
	}

	// 3. Managed Identity
	miCred, err := azidentity.NewManagedIdentityCredential(nil)
	if err == nil {
		creds = append(creds, miCred)
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("no Azure credentials available")
	}

	return azidentity.NewChainedTokenCredential(creds, nil)
}

func filter(f *fetcher, in io.Reader, out io.Writer) error {
	t := template.New(".env").Funcs(template.FuncMap{
		"kv": f.fetch,
	})
	scanner := bufio.NewScanner(in)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line != "" {
			parsed, err := t.Parse(line)
			if err != nil {
				return fmt.Errorf("line %d: failed to parse template %q: %w", lineNum, line, err)
			}
			if err := parsed.Execute(out, nil); err != nil {
				return fmt.Errorf("line %d: failed to process %q: %w", lineNum, line, err)
			}
		}
		out.Write([]byte{'\n'})
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	return nil
}

func (f *fetcher) fetch(rawurl string) (string, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(parsedURL.Hostname(), "vault.azure.net") {
		return "", fmt.Errorf("invalid url - %s", rawurl)
	}

	vaultURL := fmt.Sprintf("https://%s", parsedURL.Host)
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "secrets" {
		return "", fmt.Errorf("invalid secret URL format: %s", rawurl)
	}
	secretName := pathParts[1]
	version := ""
	if len(pathParts) >= 3 {
		version = pathParts[2]
	}

	client, err := f.getClient(vaultURL)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetSecret(ctx, secretName, version, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has nil value", secretName)
	}

	return *resp.Value, nil
}

func (f *fetcher) getClient(vaultURL string) (secretClient, error) {
	if client, ok := f.clients[vaultURL]; ok {
		return client, nil
	}

	client, err := f.factory(vaultURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %w", vaultURL, err)
	}

	f.clients[vaultURL] = client
	return client, nil
}
