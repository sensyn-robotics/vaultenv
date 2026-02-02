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
	filter(f, r, &b)
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
	filter(f, r, &b)
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
	defer func() {
		if recover() == nil {
			t.Fatalf("must panic for invalid URL")
		}
	}()
	filter(f, r, &b)
}

func TestInvalidSecretPath(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/keys/mykey" }}
`
	f := newTestFetcher(map[string]string{})
	r := strings.NewReader(template)
	defer func() {
		if recover() == nil {
			t.Fatalf("must panic for invalid secret path")
		}
	}()
	filter(f, r, &b)
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
	filter(f, r, &b)
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
	defer func() {
		if panicVal := recover(); panicVal == nil {
			t.Fatalf("expected panic due to error, but none occurred")
		}
	}()
	filter(f, r, &b)
}

func TestSecretNotFound(t *testing.T) {
	var b bytes.Buffer
	template := `PASSWORD={{ kv "https://example.vault.azure.net/secrets/nonexistent" }}
`
	f := newTestFetcher(map[string]string{
		"other": "value",
	})
	r := strings.NewReader(template)
	defer func() {
		if recover() == nil {
			t.Fatalf("must panic for secret not found")
		}
	}()
	filter(f, r, &b)
}
