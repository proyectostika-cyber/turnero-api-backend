package services

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/unknowncode44/appointments/internal/api/dto"
	db "github.com/unknowncode44/appointments/internal/db/sqlc"
	"github.com/unknowncode44/appointments/internal/repositories"
)

func mapTenant(v db.Tenant) dto.TenantResponse {
	return dto.TenantResponse{
		ID:              v.ID,
		Name:            v.Name,
		Timezone:        v.Timezone,
		Active:          v.Active,
		GreetingMessage: v.GreetingMessage,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

func mapProvider(v db.Provider) dto.ProviderResponse {
	return dto.ProviderResponse(v)
}

func mapService(v db.Service) dto.ServiceResponse {
	return dto.ServiceResponse(v)
}

func mapCustomer(v db.Customer) dto.CustomerResponse {
	return dto.CustomerResponse{
		ID:        v.ID,
		TenantID:  v.TenantID,
		FirstName: repositories.TextPtr(v.FirstName),
		LastName:  repositories.TextPtr(v.LastName),
		Notes:     repositories.TextPtr(v.Notes),
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

func mapTenantChannel(v db.TenantChannel) dto.TenantChannelResponse {
	return dto.TenantChannelResponse{
		ID:          v.ID,
		TenantID:    v.TenantID,
		ChannelType: v.ChannelType,
		ExternalID:  v.ExternalID,
		ExternalKey: repositories.TextPtr(v.ExternalKey),
		Active:      v.Active,
		CreatedAt:   v.CreatedAt,
	}
}

func mapAvailability(v db.ProviderAvailability) dto.AvailabilityResponse {
	// Convert pgtype.Time (microseconds since midnight) to string HH:MM:SS
	startTime := pgTimeToString(v.StartTime)
	endTime := pgTimeToString(v.EndTime)
	
	return dto.AvailabilityResponse{
		ID:         v.ID,
		ProviderID: v.ProviderID,
		Weekday:    v.Weekday,
		StartTime:  startTime,
		EndTime:    endTime,
	}
}

// pgTimeToString converts pgtype.Time (microseconds since midnight) to HH:MM:SS string
func pgTimeToString(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	seconds := t.Microseconds / 1_000_000
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func mapException(v db.ProviderException) dto.ExceptionResponse {
	return dto.ExceptionResponse{
		ID:         v.ID,
		ProviderID: v.ProviderID,
		StartAt:    v.StartAt,
		EndAt:      v.EndAt,
		Reason:     repositories.TextPtr(v.Reason),
		CreatedAt:  v.CreatedAt,
	}
}

func mapSlot(v db.AppointmentSlot) dto.SlotResponse {
	var appointmentID *uuid.UUID
	if v.AppointmentID.Valid {
		id := v.AppointmentID.Bytes
		parsedID, _ := uuid.FromBytes(id[:])
		appointmentID = &parsedID
	}
	return dto.SlotResponse{
		ID:            v.ID,
		TenantID:      v.TenantID,
		ProviderID:    v.ProviderID,
		StartAt:       v.StartAt,
		EndAt:         v.EndAt,
		Status:        v.Status,
		AppointmentID: appointmentID,
		CreatedAt:     v.CreatedAt,
	}
}

func mapAppointment(v db.Appointment) dto.AppointmentResponse {
	var slotID *uuid.UUID
	if v.SlotID.Valid {
		id := v.SlotID.Bytes
		parsedID, _ := uuid.FromBytes(id[:])
		slotID = &parsedID
	}
	return dto.AppointmentResponse{
		ID:         v.ID,
		TenantID:   v.TenantID,
		CustomerID: v.CustomerID,
		ProviderID: v.ProviderID,
		ServiceID:  v.ServiceID,
		SlotID:     slotID,
		StartAt:    v.StartAt,
		EndAt:      v.EndAt,
		Status:     v.Status,
		Notes:      repositories.TextPtr(v.Notes),
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
}

func mapThread(v db.ConversationThread) dto.ConversationThreadResponse {
	return dto.ConversationThreadResponse{
		ID:         v.ID,
		TenantID:   v.TenantID,
		CustomerID: v.CustomerID,
		CreatedAt:  v.CreatedAt,
	}
}

func mapMessage(v db.ConversationMessage) dto.ConversationMessageResponse {
	return dto.ConversationMessageResponse{
		ID:        v.ID,
		ThreadID:  v.ThreadID,
		Direction: v.Direction,
		Message:   v.Message,
		Metadata:  json.RawMessage(v.Metadata),
		CreatedAt: v.CreatedAt,
	}
}
