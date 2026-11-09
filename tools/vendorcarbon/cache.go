package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type sourceCacheTransport struct {
	directory string
}

func (transport sourceCacheTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	hash := sha256.Sum256([]byte(request.URL.String()))
	filename := filepath.Join(transport.directory, hex.EncodeToString(hash[:]))
	if _, err := os.Stat(filename + ".notfound"); err == nil {
		return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(nil)), Request: request}, nil
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("source cache miss for %s: %w", request.URL, err)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(content)), ContentLength: int64(len(content)), Request: request}, nil
}
