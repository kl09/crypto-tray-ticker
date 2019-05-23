package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

type SavedData struct {
	Tokens []*Token `json:"tokens"`
	Source string   `json:"source"`
}

func save(fileName string, tokens []*Token, source string) {
	data := SavedData{
		Tokens: tokens,
		Source: source,
	}

	b, _ := json.Marshal(data)
	_ = ioutil.WriteFile(fileName, b, 0666)
}

func load(fileName string, clickedTokens chan []*Token, source chan string) {
	data := SavedData{}

	b, _ := ioutil.ReadFile(fileName)
	_ = json.Unmarshal(b, &data)

	source <- data.Source
	clickedTokens <- data.Tokens
}

func makeRequest(ctx context.Context, client *http.Client, requestMethod string, requestURL string, requestBody []byte,
	headers map[string]string) (*http.Response, error) {
	var (
		body     io.Reader
		request  *http.Request
		response *http.Response
		err      error
	)

	if requestMethod == "GET" {
		body = nil
	} else {
		body = bytes.NewReader(requestBody)
	}

	if request, err = http.NewRequest(requestMethod, requestURL, body); err != nil {
		return nil, err
	}

	request = request.WithContext(ctx)
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	if response, err = client.Do(request); err != nil {
		return nil, err
	}

	return response, err
}
