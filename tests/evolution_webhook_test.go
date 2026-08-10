package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/unknowncode44/appointments/internal/api/dto"
	db "github.com/unknowncode44/appointments/internal/db/sqlc"
	"github.com/unknowncode44/appointments/internal/repositories"
	"github.com/unknowncode44/appointments/internal/services"
)

type evolutionFixture struct {
	pool        *pgxpool.Pool
	service     services.ConversationService
	tenantID    uuid.UUID
	tenantName  string
	greeting    string
	channelID   uuid.UUID
	channel     string
	customerJID string
}

type evolutionCounts struct {
	logs             int
	customers        int
	customerChannels int
	threads          int
	messages         int
}

func newEvolutionFixture(t *testing.T) evolutionFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set - skipping Evolution webhook integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := db.NewStore(pool)
	suffix := uuid.NewString()
	tenantName := "Evolution Test " + suffix
	greeting := "Test greeting " + suffix
	tenant, err := store.CreateTenant(ctx, db.CreateTenantParams{
		Name:     tenantName,
		Timezone: "UTC",
		Column3:  greeting,
	})
	require.NoError(t, err)
	channel, err := store.CreateTenantChannel(ctx, db.CreateTenantChannelParams{
		TenantID:    tenant.ID,
		ChannelType: "whatsapp",
		ExternalID:  "evolution-" + suffix,
		ExternalKey: pgtype.Text{},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM webhook_logs WHERE tenant_id = $1", tenant.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM conversation_messages WHERE thread_id IN (SELECT id FROM conversation_threads WHERE tenant_id = $1)", tenant.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM conversation_state WHERE tenant_id = $1", tenant.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM conversation_threads WHERE tenant_id = $1", tenant.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM customer_channels WHERE tenant_channel_id = $1", channel.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM customers WHERE tenant_id = $1", tenant.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM tenant_channels WHERE id = $1", channel.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenant.ID)
	})

	return evolutionFixture{
		pool:        pool,
		service:     services.NewConversationService(repositories.NewWorkflowRepository(store)),
		tenantID:    tenant.ID,
		tenantName:  tenantName,
		greeting:    greeting,
		channelID:   channel.ID,
		channel:     channel.ExternalID,
		customerJID: suffix + "@s.whatsapp.net",
	}
}

func (f evolutionFixture) process(t *testing.T, messageID string) dto.EvolutionWebhookResponse {
	t.Helper()
	result, err := f.service.ProcessEvolutionWebhook(context.Background(), dto.EvolutionWebhookRequest{
		Instance:  f.channel,
		MessageID: messageID,
		Data: dto.EvolutionData{
			Key: dto.EvolutionKey{RemoteJid: f.customerJID},
			Message: dto.EvolutionMessageBody{
				Conversation: "test message",
			},
		},
	}, []byte(`{"test":true}`))
	require.NoError(t, err)
	return result
}

func (f evolutionFixture) counts(t *testing.T) evolutionCounts {
	t.Helper()
	var counts evolutionCounts
	err := f.pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM webhook_logs WHERE tenant_id = $1),
			(SELECT count(*) FROM customers WHERE tenant_id = $1),
			(SELECT count(*) FROM customer_channels cc JOIN tenant_channels tc ON tc.id = cc.tenant_channel_id WHERE tc.tenant_id = $1),
			(SELECT count(*) FROM conversation_threads WHERE tenant_id = $1),
			(SELECT count(*) FROM conversation_messages cm JOIN conversation_threads ct ON ct.id = cm.thread_id WHERE ct.tenant_id = $1)
	`, f.tenantID).Scan(&counts.logs, &counts.customers, &counts.customerChannels, &counts.threads, &counts.messages)
	require.NoError(t, err)
	return counts
}

func TestEvolutionWebhook_FirstAndSequentialDuplicate(t *testing.T) {
	f := newEvolutionFixture(t)

	first := f.process(t, "message-1")
	require.False(t, first.Idempotent)
	require.Equal(t, evolutionCounts{logs: 1, customers: 1, customerChannels: 1, threads: 1, messages: 1}, f.counts(t))

	duplicate := f.process(t, "message-1")
	require.True(t, duplicate.Idempotent)
	require.True(t, duplicate.Processed)
	require.Equal(t, f.tenantID, duplicate.TenantID)
	require.Equal(t, f.tenantName, duplicate.TenantName)
	require.Equal(t, f.greeting, duplicate.GreetingMessage)
	require.Equal(t, first.CustomerID, duplicate.CustomerID)
	require.Equal(t, "message_received", duplicate.ConversationState.CurrentStep)
	require.NotEmpty(t, duplicate.ConversationState.Data)
	require.Equal(t, evolutionCounts{logs: 1, customers: 1, customerChannels: 1, threads: 1, messages: 1}, f.counts(t))
}

func TestEvolutionWebhook_ConcurrentDuplicate(t *testing.T) {
	f := newEvolutionFixture(t)
	const deliveries = 8

	results := make(chan dto.EvolutionWebhookResponse, deliveries)
	errs := make(chan error, deliveries)
	var wg sync.WaitGroup
	for range deliveries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := f.service.ProcessEvolutionWebhook(context.Background(), dto.EvolutionWebhookRequest{
				Instance:  f.channel,
				MessageID: "concurrent-message",
				Data: dto.EvolutionData{
					Key:     dto.EvolutionKey{RemoteJid: f.customerJID},
					Message: dto.EvolutionMessageBody{Conversation: "test message"},
				},
			}, []byte(`{"test":true}`))
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		require.NoError(t, err)
	}

	var newCount, duplicateCount int
	for result := range results {
		if result.Idempotent {
			duplicateCount++
		} else {
			newCount++
		}
	}
	require.Equal(t, 1, newCount)
	require.Equal(t, deliveries-1, duplicateCount)
	require.Equal(t, evolutionCounts{logs: 1, customers: 1, customerChannels: 1, threads: 1, messages: 1}, f.counts(t))
}

func TestEvolutionWebhook_IdempotencyScope(t *testing.T) {
	t.Run("same ID on different channels", func(t *testing.T) {
		first := newEvolutionFixture(t)
		second := newEvolutionFixture(t)
		require.False(t, first.process(t, "shared-message").Idempotent)
		require.False(t, second.process(t, "shared-message").Idempotent)
		require.Equal(t, evolutionCounts{logs: 1, customers: 1, customerChannels: 1, threads: 1, messages: 1}, first.counts(t))
		require.Equal(t, evolutionCounts{logs: 1, customers: 1, customerChannels: 1, threads: 1, messages: 1}, second.counts(t))
	})

	t.Run("different IDs on same channel", func(t *testing.T) {
		f := newEvolutionFixture(t)
		require.False(t, f.process(t, "message-1").Idempotent)
		require.False(t, f.process(t, "message-2").Idempotent)
		require.Equal(t, evolutionCounts{logs: 2, customers: 1, customerChannels: 1, threads: 1, messages: 2}, f.counts(t))
	})
}

func TestEvolutionWebhook_MissingAndBlankIDsUseLegacyProcessing(t *testing.T) {
	for _, messageID := range []string{"", "   "} {
		t.Run(fmt.Sprintf("message ID %q", messageID), func(t *testing.T) {
			f := newEvolutionFixture(t)
			require.False(t, f.process(t, messageID).Idempotent)
			require.False(t, f.process(t, messageID).Idempotent)
			require.Equal(t, evolutionCounts{logs: 2, customers: 1, customerChannels: 1, threads: 1, messages: 2}, f.counts(t))
		})
	}
}
