package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/unknowncode44/appointments/internal/api/response"
	"github.com/unknowncode44/appointments/internal/util"
)

// EvolutionSendTextResult contains the Evolution API response data.
type EvolutionSendTextResult struct {
	StatusCode  int
	Body        []byte
	ContentType string
}

type EvolutionService interface {
	SendText(ctx context.Context, instance, number, text string) (EvolutionSendTextResult, error)
}

type evolutionService struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewEvolutionService creates a new Evolution service using configured values.
func NewEvolutionService(config util.Config) EvolutionService {
	return newEvolutionService(config.EvoAPIURL, config.EvoAPIKey, http.DefaultClient)
}

func newEvolutionService(baseURL, apiKey string, client *http.Client) EvolutionService {
	return &evolutionService{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  client,
	}
}

func (s *evolutionService) SendText(ctx context.Context, instance, number, text string) (EvolutionSendTextResult, error) {
	if strings.TrimSpace(instance) == "" || strings.TrimSpace(number) == "" || strings.TrimSpace(text) == "" {
		return EvolutionSendTextResult{}, response.ErrInvalidInput
	}
	if s.baseURL == "" || s.apiKey == "" {
		return EvolutionSendTextResult{}, errors.New("evolution api not configured")
	}
	payload := map[string]string{
		"number": number,
		"text":   text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return EvolutionSendTextResult{}, err
	}
	url := s.baseURL + "/message/sendText/" + instance
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return EvolutionSendTextResult{}, err
	}
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return EvolutionSendTextResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return EvolutionSendTextResult{}, err
	}

	return EvolutionSendTextResult{
		StatusCode:  resp.StatusCode,
		Body:        respBody,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}
