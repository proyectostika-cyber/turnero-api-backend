package dto

import (
	"encoding/json"
	"testing"
)

func TestEvolutionWebhookRequestResolvedMessageID(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "flat message ID",
			body: `{"messageId":"flat-id"}`,
			want: "flat-id",
		},
		{
			name: "nested message ID",
			body: `{"data":{"key":{"id":"nested-id"}}}`,
			want: "nested-id",
		},
		{
			name: "nested message ID takes precedence",
			body: `{"messageId":"flat-id","data":{"key":{"id":"nested-id"}}}`,
			want: "nested-id",
		},
		{
			name: "missing message ID",
			body: `{}`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req EvolutionWebhookRequest
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatalf("unmarshal webhook request: %v", err)
			}
			if got := req.ResolvedMessageID(); got != tt.want {
				t.Fatalf("ResolvedMessageID() = %q, want %q", got, tt.want)
			}
		})
	}
}
