package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

type dummyClient struct{}

func (c *dummyClient) Do(req *http.Request) (*http.Response, error) {
	var body string
	if strings.HasPrefix(req.URL.String(), "http://169.254.169.254") {
		body = `{
  "access_token": "TOKEN_WITH_VM_IDENTITY",
  "refresh_token": "",
  "expires_in": "3599",
  "expires_on": "1506484173",
  "not_before": "1506480273",
  "resource": "https://vault.azure.net/",
  "token_type": "Bearer"
}`
	} else if strings.HasPrefix(req.URL.String(), "https://login.microsoftonline.com") {
		body = `{
  "access_token": "TOKEN_WITH_CLIENT_CREDENTIAL",
  "refresh_token": "",
  "expires_in": "3599",
  "expires_on": "1506484173",
  "not_before": "1506480273",
  "resource": "https://vault.azure.net/",
  "token_type": "Bearer"
}`
	} else if req.Header.Get("Authorization") == "Bearer TOKEN_WITH_VM_IDENTITY" {
		body = `{
  "value": "mysecretvalue1",
  "id": "https://example.vault.azure.net/secrets/pass/4387e9f3d6e14c459867679a90fd0f79",
  "attributes": {
    "enabled": true,
    "created": 1493938410,
    "updated": 1493938410,
    "recoveryLevel": "Recoverable+Purgeable"
  }
}`
	} else if req.Header.Get("Authorization") == "Bearer TOKEN_WITH_CLIENT_CREDENTIAL" {
		body = `{
  "value": "mysecretvalue2",
  "id": "https://example.vault.azure.net/secrets/pass/4387e9f3d6e14c459867679a90fd0f79",
  "attributes": {
    "enabled": true,
    "created": 1493938410,
    "updated": 1493938410,
    "recoveryLevel": "Recoverable+Purgeable"
  }
}`
	} else {
		return nil, errors.New("Unexpected request")
	}
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

func TestValidTemplateWithVmIdentity(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass" }}
`
	expected := `USER=foo@example.com
PASSWORD=mysecretvalue1
`
	client := &dummyClient{}
	r := strings.NewReader(template)
	filter(fetcher{client, ""}, r, &b)
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}

func TestValidTemplateWithClientCredential(t *testing.T) {
	os.Setenv("VAULTENV_AZURE_USER", "b3a0fa1e-2a56-44c5-9ec1-f95921243ed7")
	os.Setenv("VAULTENV_AZURE_PASSWORD", "7a724b98-f30e-4991-a020-fb56d12277e1")
	os.Setenv("VAULTENV_AZURE_TENANT", "5a9c134c-c9d6-4b9c-b588-94d3096dbf4c")

	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass" }}
`
	expected := `USER=foo@example.com
PASSWORD=mysecretvalue2
`
	client := &dummyClient{}
	r := strings.NewReader(template)
	filter(fetcher{client, ""}, r, &b)
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}

func TestInvalidUrl(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://invalid.sensyn.net/secrets/pass" }}
`
	client := &dummyClient{}
	r := strings.NewReader(template)
	defer func() {
		recover()
	}()
	filter(fetcher{client, ""}, r, &b)
	t.Fatalf("must be panic")
}

func TestEmptyLine(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com

PASSWORD=mysecretvalue1
`
	expected := `USER=foo@example.com

PASSWORD=mysecretvalue1
`
	client := &dummyClient{}
	r := strings.NewReader(template)
	filter(fetcher{client, ""}, r, &b)
	if b.String() != expected {
		t.Fatalf("got:%s want:%s", b.String(), expected)
	}
}
