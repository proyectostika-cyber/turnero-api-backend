package repositories

import (
	db "github.com/unknowncode44/appointments/internal/db/sqlc"
)

// Tenant row mappers
func ToTenant(row db.GetTenantRow) db.Tenant {
	return db.Tenant{
		ID:              row.ID,
		Name:            row.Name,
		Timezone:        row.Timezone,
		Active:          row.Active,
		GreetingMessage: row.GreetingMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

// toTenant is an alias for backwards compatibility
func toTenant(row db.GetTenantRow) db.Tenant {
	return ToTenant(row)
}

func toTenantFromCreate(row db.CreateTenantRow) db.Tenant {
	return db.Tenant{
		ID:              row.ID,
		Name:            row.Name,
		Timezone:        row.Timezone,
		Active:          row.Active,
		GreetingMessage: row.GreetingMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toTenantFromUpdate(row db.UpdateTenantRow) db.Tenant {
	return db.Tenant{
		ID:              row.ID,
		Name:            row.Name,
		Timezone:        row.Timezone,
		Active:          row.Active,
		GreetingMessage: row.GreetingMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toTenantFromDeactivate(row db.DeactivateTenantRow) db.Tenant {
	return db.Tenant{
		ID:              row.ID,
		Name:            row.Name,
		Timezone:        row.Timezone,
		Active:          row.Active,
		GreetingMessage: row.GreetingMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toTenantFromUpdateGreeting(row db.UpdateTenantGreetingRow) db.Tenant {
	return db.Tenant{
		ID:              row.ID,
		Name:            row.Name,
		Timezone:        row.Timezone,
		Active:          row.Active,
		GreetingMessage: row.GreetingMessage,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toTenants(rows []db.ListTenantsRow) []db.Tenant {
	tenants := make([]db.Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = db.Tenant{
			ID:              row.ID,
			Name:            row.Name,
			Timezone:        row.Timezone,
			Active:          row.Active,
			GreetingMessage: row.GreetingMessage,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
	}
	return tenants
}

// CountTenantsParams from ListTenantsParams
func toCountTenantsParams(arg db.ListTenantsParams) db.CountTenantsParams {
	return db.CountTenantsParams{
		Column1: arg.Column1,
		Column2: arg.Column2,
	}
}

// CountProvidersParams from ListProvidersParams
func toCountProvidersParams(arg db.ListProvidersParams) db.CountProvidersParams {
	return db.CountProvidersParams{
		TenantID: arg.TenantID,
		Column2:  arg.Column2,
		Column3:  arg.Column3,
	}
}

// CountServicesParams from ListServicesParams
func toCountServicesParams(arg db.ListServicesParams) db.CountServicesParams {
	return db.CountServicesParams{
		TenantID: arg.TenantID,
		Column2:  arg.Column2,
		Column3:  arg.Column3,
	}
}

// CountCustomersParams from ListCustomersParams
func toCountCustomersParams(arg db.ListCustomersParams) db.CountCustomersParams {
	return db.CountCustomersParams{
		TenantID: arg.TenantID,
		Column2:  arg.Column2,
	}
}

// CountTenantChannelsParams from ListTenantChannelsParams
func toCountTenantChannelsParams(arg db.ListTenantChannelsParams) db.CountTenantChannelsParams {
	return db.CountTenantChannelsParams{
		TenantID: arg.TenantID,
		Column2:  arg.Column2,
		Column3:  arg.Column3,
	}
}

// CountAppointmentsParams from ListAppointmentsParams
func toCountAppointmentsParams(arg db.ListAppointmentsParams) db.CountAppointmentsParams {
	return db.CountAppointmentsParams{
		TenantID: arg.TenantID,
		Column2:  arg.Column2,
		Column3:  arg.Column3,
		Column4:  arg.Column4,
	}
}
