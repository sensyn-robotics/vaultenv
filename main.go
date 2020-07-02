package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
type fetcher struct {
	client httpClient
	token  string
}

func main() {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	filter(fetcher{client, ""}, os.Stdin, os.Stdout)
}

func filter(f fetcher, in io.Reader, out io.Writer) {
	t := template.New(".env").Funcs(template.FuncMap{
		"kv": f.fetch,
	})
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		line := scanner.Text()
		if line != "" {
			err := template.Must(t.Parse(line)).Execute(out, nil)
			if err != nil {
				panic(err)
			}
		}
		out.Write([]byte{'\n'})
	}
}

func (f *fetcher) fetch(rawurl string) (string, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(url.Hostname(), "vault.azure.net") {
		return "", fmt.Errorf("Invalid url - %s", rawurl)
	}
	b, err := f.getToken()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", rawurl+"?api-version=7.0", nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+b)
	req.Header.Add("Accept", "application/json")
	res, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("GET %s - %s", url, res.Status)
	}
	defer res.Body.Close()
	var result struct {
		Value string `json:"value"`
	}
	decoder := json.NewDecoder(res.Body)
	if err = decoder.Decode(&result); err != nil {
		return "", err
	}

	return result.Value, nil
}

func (f *fetcher) getToken() (string, error) {
	if f.token != "" {
		return f.token, nil
	}
	var req *http.Request
	if clientId := os.Getenv("AZURE_USER"); clientId != "" {
		values := url.Values{}
		values.Set("grant_type", "client_credentials")
		values.Add("client_id", clientId)
		values.Add("client_secret", os.Getenv("AZURE_PASSWORD"))
		values.Add("resource", "https://vault.azure.net")
		req, _ = http.NewRequest("GET", fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", os.Getenv("AZURE_TENANT")), strings.NewReader(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, _ = http.NewRequest("GET", "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2019-06-04&resource=https%3A%2F%2Fvault.azure.net", nil)
		req.Header.Add("Metadata", "true")
	}
	res, err := f.client.Do(req)
	if res.StatusCode != 200 {
		return "", errors.New(res.Status)
	}
	defer res.Body.Close()
	var auth struct {
		Token string `json:"access_token"`
	}
	decoder := json.NewDecoder(res.Body)
	if err = decoder.Decode(&auth); err != nil {
		return "", err
	}
	f.token = auth.Token

	return auth.Token, nil
}
