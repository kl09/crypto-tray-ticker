package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

type MockTransport struct {
}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response := &http.Response{
		Header:     make(http.Header),
		Request:    req,
		StatusCode: http.StatusOK,
	}

	if req.URL.Path == "/v2/assets" {
		response.Body, _ = os.Open("./tests/assets.json")
	}

	return response, nil
}

func TestApp_GetTokens(t *testing.T) {
	client := &http.Client{
		Transport: &MockTransport{},
		Timeout:   10 * time.Second,
	}

	app := &App{
		client: client,
	}
	tokens, err := app.getTokens()
	require.NoError(t, err)

	if assert.Len(t, tokens, 50) {
		assert.Equal(t, "bitcoin", tokens[0].ID)
		assert.Equal(t, "BTC", tokens[0].Symbol)
		assert.Equal(t, "Bitcoin", tokens[0].Name)
		assert.Equal(t, json.Number("7791.7934691770748761"), tokens[0].PriceUsd)
		assert.Equal(t, json.Number("-1.4691508362430571"), tokens[0].ChangePercent24Hr)
	}

}
