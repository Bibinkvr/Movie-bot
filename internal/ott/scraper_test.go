package ott

import (
	"context"
	"testing"
)

func TestMakeTMDBRequest(t *testing.T) {
	ctx := context.Background()

	// 1. Test standard v3 API Key (32 characters or less)
	key32 := "12345678901234567890123456789012"
	url32 := tmdbBase + "/search/movie?api_key=" + key32 + "&query=test"
	req32, err := makeTMDBRequest(ctx, "GET", url32, key32)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if req32.URL.Query().Get("api_key") != key32 {
		t.Errorf("Expected api_key query param to match key32, got: %s", req32.URL.Query().Get("api_key"))
	}
	if req32.Header.Get("Authorization") != "" {
		t.Errorf("Expected Authorization header to be empty, got: %s", req32.Header.Get("Authorization"))
	}

	// 2. Test v4 Access Token (JWT - greater than 32 characters)
	keyJWT := "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiI3Yjg4NmRiMTdjNTRlZGIxNmQxYzFiNTE0MzRlZWYyYiIsIm5iZiI6MTc2NTY0Mzc5Ni41NTIsInN1YiI6IjY5M2Q5NjE0ZDhiMDQwZTBjYTc4MGMxMSIsInNjb3BlcyI6WyJhcGlfcmVhZCJdLCJ2ZXJzaW9uIjoxfQ.StiLGAuVT-yYd6uaFW-f0kI735ChL1MjWuejFjSoHng"
	urlJWT := tmdbBase + "/search/movie?api_key=" + keyJWT + "&query=test"
	reqJWT, err := makeTMDBRequest(ctx, "GET", urlJWT, keyJWT)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if reqJWT.URL.Query().Get("api_key") != "" {
		t.Errorf("Expected api_key query param to be deleted, got: %s", reqJWT.URL.Query().Get("api_key"))
	}
	expectedAuthHeader := "Bearer " + keyJWT
	if reqJWT.Header.Get("Authorization") != expectedAuthHeader {
		t.Errorf("Expected Authorization header to be %s, got: %s", expectedAuthHeader, reqJWT.Header.Get("Authorization"))
	}
}
