package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type mockSecretClient struct {
	secrets map[string]string
}

func (m *mockSecretClient) GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	value, ok := m.secrets[name]
	if !ok {
		return azsecrets.GetSecretResponse{}, errors.New("secret not found")
	}
	return azsecrets.GetSecretResponse{
		Secret: azsecrets.Secret{
			Value: &value,
		},
	}, nil
}

type errorSecretClient struct{}

func (e *errorSecretClient) GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	return azsecrets.GetSecretResponse{}, errors.New("connection timeout")
}

func newTestFetcher(secrets map[string]string) *fetcher {
	client := &mockSecretClient{secrets: secrets}
	return &fetcher{
		factory: func(vaultURL string) (secretClient, error) {
			return client, nil
		},
		clients: make(map[string]secretClient),
	}
}

func newErrorFetcher() *fetcher {
	return &fetcher{
		factory: func(vaultURL string) (secretClient, error) {
			return &errorSecretClient{}, nil
		},
		clients: make(map[string]secretClient),
	}
}

func TestValidTemplate(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass" }}
`
	expected := `USER=foo@example.com
PASSWORD=mysecretvalue
`
	f := newTestFetcher(map[string]string{
		"pass": "mysecretvalue",
	})
	r := strings.NewReader(template)
	if err := filter(f, r, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}

func TestValidTemplateWithVersion(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass/abc123" }}
`
	expected := `PASSWORD=mysecretvalue
`
	f := newTestFetcher(map[string]string{
		"pass": "mysecretvalue",
	})
	r := strings.NewReader(template)
	if err := filter(f, r, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}

func TestInvalidUrl(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://invalid.sensyn.net/secrets/pass" }}
`
	f := newTestFetcher(map[string]string{})
	r := strings.NewReader(template)
	err := filter(f, r, &b)
	if err == nil {
		t.Fatalf("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid url") {
		t.Fatalf("expected 'invalid url' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("expected line number in error, got: %v", err)
	}
}

func TestInvalidSecretPath(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/keys/mykey" }}
`
	f := newTestFetcher(map[string]string{})
	r := strings.NewReader(template)
	err := filter(f, r, &b)
	if err == nil {
		t.Fatalf("expected error for invalid secret path")
	}
	if !strings.Contains(err.Error(), "invalid secret URL format") {
		t.Fatalf("expected 'invalid secret URL format' in error, got: %v", err)
	}
}

func TestEmptyLine(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com

PASSWORD=mysecretvalue
`
	expected := `USER=foo@example.com

PASSWORD=mysecretvalue
`
	f := newTestFetcher(map[string]string{})
	r := strings.NewReader(template)
	if err := filter(f, r, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}

func TestSecretClientError(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass" }}
`
	f := newErrorFetcher()
	r := strings.NewReader(template)
	err := filter(f, r, &b)
	if err == nil {
		t.Fatalf("expected error due to client error")
	}
	if !strings.Contains(err.Error(), "connection timeout") {
		t.Fatalf("expected 'connection timeout' in error, got: %v", err)
	}
}

func TestSecretNotFound(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/secrets/nonexistent" }}
`
	f := newTestFetcher(map[string]string{
		"other": "value",
	})
	r := strings.NewReader(template)
	err := filter(f, r, &b)
	if err == nil {
		t.Fatalf("expected error for secret not found")
	}
	if !strings.Contains(err.Error(), "secret not found") {
		t.Fatalf("expected 'secret not found' in error, got: %v", err)
	}
}
