package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type dummyClient struct{}

func (c *dummyClient) Do(req *http.Request) (*http.Response, error) {
	var body string
	if strings.HasPrefix(req.URL.String(), "http://169.254.169.254") {
		body = `{
  "access_token": "eyJ0eXAiYzU5NTQ4YzNjNTc2MjI4NDg2YTFmMDAzN2ViMTZhMWIK",
  "refresh_token": "",
  "expires_in": "3599",
  "expires_on": "1506484173",
  "not_before": "1506480273",
  "resource": "https://vault.azure.net/",
  "token_type": "Bearer"
}`
	} else {
		body = `{
  "value": "mysecretvalue",
  "id": "https://example.vault.azure.net/secrets/pass/4387e9f3d6e14c459867679a90fd0f79",
  "attributes": {
    "enabled": true,
    "created": 1493938410,
    "updated": 1493938410,
    "recoveryLevel": "Recoverable+Purgeable"
  }
}`
	}
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

func TestValidTemplate(t *testing.T) {
	var b bytes.Buffer
	template := `USER=foo@example.com
PASSWORD={{ kv "https://example.vault.azure.net/secrets/pass" }}
`
	expected := `USER=foo@example.com
PASSWORD=mysecretvalue
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
