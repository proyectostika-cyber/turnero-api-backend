package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvolutionServiceSendText(t *testing.T) {
	expectedInstance := "test-instance"
	expectedNumber := "+5491112345678"
	expectedText := "hola"
	expectedAPIKey := "evo-key-123"

	var gotPath string
	var gotHeaders http.Header
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		gotBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}))
	defer server.Close()

	svc := newEvolutionService(server.URL, expectedAPIKey, server.Client())
	result, err := svc.SendText(context.Background(), expectedInstance, expectedNumber, expectedText)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, result.StatusCode)
	require.Equal(t, "application/json", result.ContentType)
	require.JSONEq(t, `{"status":"queued"}`, string(result.Body))

	require.Equal(t, "/message/sendText/"+expectedInstance, gotPath)
	require.Equal(t, expectedAPIKey, gotHeaders.Get("apikey"))
	require.Equal(t, "application/json", gotHeaders.Get("Content-Type"))
	require.JSONEq(t, fmt.Sprintf(`{"number":"%s","text":"%s"}`, expectedNumber, expectedText), string(gotBody))
}
