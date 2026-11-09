package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestImports(t *testing.T) {
	source := []byte(`import{a}from"./a.js";import './side.js';export{b}from'./b.js';export * from "./c.js";import("./dynamic.js");`)
	want := []string{"./a.js", "./b.js", "./c.js", "./dynamic.js", "./side.js"}
	if got := imports(source); !reflect.DeepEqual(got, want) {
		t.Fatalf("imports: got %v, want %v", got, want)
	}
}

func TestSizeSpecificArtworkFallsBackOnlyWhenMissing(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.Path)
		if request.URL.Path == "/svg/16/close.svg" {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = response.Write([]byte(`<svg viewBox="0 0 32 32"><path d="M0 0"/></svg>`))
	}))
	defer server.Close()
	files := []download{{Name: "icons/carbon/16/close.svg", Source: server.URL + "/svg/16/close.svg", Fallback: server.URL + "/svg/32/close.svg"}}
	fetch(server.Client(), files)
	if files[0].Err != nil || files[0].Source != server.URL+"/svg/32/close.svg" {
		t.Fatalf("size-specific fallback failed: %+v", files[0])
	}
	if !reflect.DeepEqual(requests, []string{"/svg/16/close.svg", "/svg/32/close.svg"}) {
		t.Fatalf("unexpected artwork requests: %v", requests)
	}
}

func TestSourceCacheIsOfflineAndKeyedByExactURL(t *testing.T) {
	directory := t.TempDir()
	source := "https://example.invalid/pinned/asset.js"
	hash := sha256.Sum256([]byte(source))
	filename := filepath.Join(directory, hex.EncodeToString(hash[:]))
	if err := os.WriteFile(filename, []byte("source bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: sourceCacheTransport{directory: directory}}
	content, err := get(client, source)
	if err != nil || string(content) != "source bytes" {
		t.Fatalf("cached source: %s, %v", content, err)
	}
	if _, err := get(client, source+"?different"); err == nil {
		t.Fatal("source cache unexpectedly used the network or an inexact URL")
	}
	if err := os.WriteFile(filename+".notfound", []byte("404"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := get(client, source); err == nil {
		t.Fatal("negative source cache did not preserve HTTP 404")
	}
}

func TestNames(t *testing.T) {
	got, err := names("tabs,button,tabs, text-input")
	if err != nil || !reflect.DeepEqual(got, []string{"button", "tabs", "text-input"}) {
		t.Fatalf("names: %v, %v", got, err)
	}
	if _, err := names("../outside"); err == nil {
		t.Fatal("accepted a path traversal")
	}
}
