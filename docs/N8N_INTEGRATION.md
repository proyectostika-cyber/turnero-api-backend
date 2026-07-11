# N8N Integration Guide

## Overview

This document describes how to integrate n8n workflows with the Appointments API using API Key authentication.

## Architecture

```
n8n Workflow → API Key Auth → Tenant Validation → Backend Handler → Database
```

### Key Features

- ✅ **API Key Authentication**: No JWT required, only `X-API-Key` header
- ✅ **Multi-tenant Support**: Dynamic `tenant_id` from headers or request body
- ✅ **Separate Routes**: All N8N routes under `/api/v1/n8n/*` prefix
- ✅ **Rate Limiting**: 500 requests/minute (vs 200 global limit)
- ✅ **CORS Enabled**: Configured for cross-origin requests

---

## Authentication

### Required Headers

All N8N integration endpoints require these headers:

```http
X-API-Key: your_n8n_api_key_here
X-Tenant-ID: tenant-uuid-here
Content-Type: application/json
```

### Getting Your API Key

The N8N API key is configured in the backend's `app.env` file:

```env
N8N_API_KEY=your_secure_api_key_here
```

Contact your backend administrator to obtain this key.

---

## Base URL

**Production:** `https://protika-bot-api-server.com/api/v1/n8n`

**Local Development:** `http://localhost:8080/api/v1/n8n`

---

## Available Endpoints

### 📋 GET Endpoints (Query Data)

#### List Services

```http
GET /api/v1/n8n/services
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Query Parameters:**
- `search` (optional): Search by name
- `active` (optional): `true` or `false`
- `page` (optional): Page number
- `per_page` (optional): Items per page

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "name": "Haircut",
      "description": "Basic haircut service",
      "duration_minutes": 30,
      "price": 25.00,
      "active": true
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 10,
    "total": 5
  }
}
```

---

#### List Providers

```http
GET /api/v1/n8n/providers
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Query Parameters:**
- `search` (optional): Search by name
- `active` (optional): `true` or `false`
- `page` (optional): Page number
- `per_page` (optional): Items per page

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "name": "John Barber",
      "active": true
    }
  ]
}
```

---

#### List Customers

```http
GET /api/v1/n8n/customers
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Query Parameters:**
- `search` (optional): Search by name/phone
- `page` (optional): Page number
- `per_page` (optional): Items per page

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "name": "Jane Doe",
      "phone": "+1234567890",
      "email": "jane@example.com"
    }
  ]
}
```

---

#### Get Availability

```http
GET /api/v1/n8n/availability
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Query Parameters (all required):**
- `provider_id`: Provider UUID
- `service_id`: Service UUID
- `date`: Date in format `YYYY-MM-DD`

**Response:**
```json
{
  "data": [
    {
      "slot_start": "2026-07-10T09:00:00Z",
      "slot_end": "2026-07-10T09:30:00Z",
      "available": true
    }
  ]
}
```

---

#### List Appointments

```http
GET /api/v1/n8n/appointments
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Query Parameters:**
- `provider_id` (optional): Filter by provider
- `customer_id` (optional): Filter by customer
- `status` (optional): `pending`, `confirmed`, `completed`, `cancelled`
- `page` (optional): Page number
- `per_page` (optional): Items per page

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "customer_id": "uuid",
      "provider_id": "uuid",
      "service_id": "uuid",
      "slot_start": "2026-07-10T10:00:00Z",
      "slot_end": "2026-07-10T10:30:00Z",
      "status": "confirmed"
    }
  ]
}
```

---

#### Get Single Appointment

```http
GET /api/v1/n8n/appointments/:id
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Response:**
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "customer_id": "uuid",
  "provider_id": "uuid",
  "service_id": "uuid",
  "slot_start": "2026-07-10T10:00:00Z",
  "slot_end": "2026-07-10T10:30:00Z",
  "status": "confirmed"
}
```

---

### ✏️ POST Endpoints (Create Data)

#### Create Customer

```http
POST /api/v1/n8n/customers
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
Content-Type: application/json
```

**Body:**
```json
{
  "tenant_id": "tenant_uuid",
  "name": "John Doe",
  "phone": "+1234567890",
  "email": "john@example.com",
  "notes": "Regular customer"
}
```

**Response:**
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "name": "John Doe",
  "phone": "+1234567890",
  "email": "john@example.com"
}
```

---

#### Create Appointment

```http
POST /api/v1/n8n/appointments
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
Content-Type: application/json
```

**Body:**
```json
{
  "tenant_id": "tenant_uuid",
  "customer_id": "customer_uuid",
  "provider_id": "provider_uuid",
  "service_id": "service_uuid",
  "slot_start": "2026-07-10T10:00:00Z",
  "slot_end": "2026-07-10T10:30:00Z",
  "notes": "First appointment"
}
```

**Response:**
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "customer_id": "uuid",
  "provider_id": "uuid",
  "service_id": "uuid",
  "slot_start": "2026-07-10T10:00:00Z",
  "slot_end": "2026-07-10T10:30:00Z",
  "status": "pending"
}
```

---

#### Process Inbound Message

```http
POST /api/v1/n8n/inbound-messages
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
Content-Type: application/json
```

**Body:**
```json
{
  "tenant_id": "tenant_uuid",
  "customer_phone": "+1234567890",
  "message": "I want to book an appointment",
  "channel": "whatsapp"
}
```

**Response:**
```json
{
  "conversation_id": "uuid",
  "message_id": "uuid",
  "status": "processed"
}
```

---

### 🔄 PATCH Endpoints (Update Data)

#### Update Appointment

```http
PATCH /api/v1/n8n/appointments/:id
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
Content-Type: application/json
```

**Body (all fields optional):**
```json
{
  "status": "confirmed",
  "slot_start": "2026-07-10T11:00:00Z",
  "slot_end": "2026-07-10T11:30:00Z",
  "notes": "Updated time"
}
```

**Response:**
```json
{
  "id": "uuid",
  "status": "confirmed",
  "slot_start": "2026-07-10T11:00:00Z",
  "slot_end": "2026-07-10T11:30:00Z"
}
```

---

### 🗑️ DELETE Endpoints

#### Cancel Appointment

```http
DELETE /api/v1/n8n/appointments/:id
```

**Headers:**
```
X-API-Key: your_api_key
X-Tenant-ID: tenant_uuid
```

**Response:**
```json
{
  "id": "uuid",
  "status": "cancelled"
}
```

---

## N8N Configuration Examples

### Example 1: HTTP Request Node (Get Services)

```json
{
  "method": "GET",
  "url": "https://protika-bot-api-server.com/api/v1/n8n/services",
  "headers": {
    "X-API-Key": "={{$env.N8N_API_KEY}}",
    "X-Tenant-ID": "={{$json.tenant_id}}"
  },
  "options": {
    "timeout": 30000
  }
}
```

### Example 2: HTTP Request Node (Create Appointment)

```json
{
  "method": "POST",
  "url": "https://protika-bot-api-server.com/api/v1/n8n/appointments",
  "headers": {
    "X-API-Key": "={{$env.N8N_API_KEY}}",
    "X-Tenant-ID": "={{$json.tenant_id}}",
    "Content-Type": "application/json"
  },
  "body": {
    "tenant_id": "={{$json.tenant_id}}",
    "customer_id": "={{$json.customer_id}}",
    "provider_id": "={{$json.provider_id}}",
    "service_id": "={{$json.service_id}}",
    "slot_start": "={{$json.slot_start}}",
    "slot_end": "={{$json.slot_end}}"
  }
}
```

### Example 3: Environment Variable Setup in n8n

In n8n settings, add this environment variable:

```
N8N_API_KEY=your_secure_api_key_here
```

Then reference it in HTTP Request nodes as: `{{$env.N8N_API_KEY}}`

---

## Error Handling

### Common Error Responses

#### 401 Unauthorized - Missing API Key

```json
{
  "error": "missing X-API-Key header"
}
```

**Solution:** Add `X-API-Key` header with valid API key.

---

#### 401 Unauthorized - Invalid API Key

```json
{
  "error": "invalid API key"
}
```

**Solution:** Verify your API key matches the one configured in backend.

---

#### 400 Bad Request - Missing Tenant ID

```json
{
  "error": "tenant_id is required (header X-Tenant-ID, query param, or body)"
}
```

**Solution:** Add `X-Tenant-ID` header or include `tenant_id` in request body.

---

#### 400 Bad Request - Invalid Tenant ID

```json
{
  "error": "invalid tenant_id format, must be valid UUID"
}
```

**Solution:** Ensure tenant_id is a valid UUID format.

---

#### 404 Not Found - Tenant Not Found

```json
{
  "error": "tenant not found"
}
```

**Solution:** Verify the tenant_id exists in the database.

---

#### 403 Forbidden - Inactive Tenant

```json
{
  "error": "tenant is not active"
}
```

**Solution:** Contact administrator to activate the tenant.

---

#### 429 Too Many Requests

```json
{
  "error": "rate limit exceeded for integration"
}
```

**Solution:** Wait before making more requests. N8N routes are limited to 500 requests per minute.

---

## Testing with cURL

### Test API Key Authentication

```bash
# Without API Key (should fail)
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services

# With API Key but no tenant (should fail)
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services \
  -H "X-API-Key: your_api_key"

# With API Key and Tenant (should succeed)
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services \
  -H "X-API-Key: your_api_key" \
  -H "X-Tenant-ID: tenant_uuid"
```

### Test Create Appointment

```bash
curl -X POST https://protika-bot-api-server.com/api/v1/n8n/appointments \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -H "X-Tenant-ID: tenant_uuid" \
  -d '{
    "tenant_id": "tenant_uuid",
    "customer_id": "customer_uuid",
    "provider_id": "provider_uuid",
    "service_id": "service_uuid",
    "slot_start": "2026-07-10T10:00:00Z",
    "slot_end": "2026-07-10T10:30:00Z"
  }'
```

---

## Security Best Practices

### For Backend Administrators

1. **API Key Security:**
   - Use strong, randomly generated API keys (minimum 32 characters)
   - Store in environment variables, never commit to git
   - Rotate keys periodically (every 90 days recommended)

2. **Tenant Isolation:**
   - Always validate tenant_id exists and is active
   - Ensure tenant data is properly isolated in database queries
   - Log all access attempts for audit trail

3. **Rate Limiting:**
   - N8N routes: 500 req/min (configured in server.go)
   - Global routes: 200 req/min
   - Adjust based on your needs

### For N8N Users

1. **Environment Variables:**
   - Store API keys in n8n environment variables
   - Never hardcode keys in workflow JSON
   - Use `{{$env.N8N_API_KEY}}` syntax

2. **Error Handling:**
   - Always check response status codes
   - Implement retry logic for 429 (rate limit) errors
   - Log errors for debugging

3. **Data Validation:**
   - Validate UUIDs before sending requests
   - Check required fields are present
   - Handle empty responses gracefully

---

## Troubleshooting

### Issue: CORS Error in Browser

**Symptom:** Browser console shows CORS policy error

**Solution:** 
- Ensure `ALLOWED_ORIGINS` in `app.env` includes your n8n domain
- Example: `ALLOWED_ORIGINS=https://n8n.hvdevs.com,https://protika-bot-api-server.com`

---

### Issue: 401 Error Even with Correct API Key

**Symptom:** Getting 401 despite correct API key

**Possible Causes:**
1. Extra spaces in header value
2. API key in backend `app.env` doesn't match
3. Backend hasn't been restarted after config change

**Solution:**
```bash
# In VPS, restart backend
cd /opt/apps/turnero-api-backend
docker compose -f docker-compose.prod.yml restart app

# Verify app.env has correct key
cat app.env | grep N8N_API_KEY
```

---

### Issue: Tenant Not Found

**Symptom:** `404 tenant not found`

**Solution:**
```bash
# Check tenant exists in database
docker compose -f docker-compose.prod.yml exec postgres psql -U appointments_user -d appointments_prod

# In psql:
SELECT id, name, active FROM tenants;
\q
```

---

## Migration from Old Routes

If you were using the old `/api/v1/integrations/*` routes:

### Old Route → New Route Mapping

| Old Route | New Route |
|-----------|-----------|
| `POST /api/v1/integrations/webhooks/evolution` | `POST /api/v1/webhooks/evolution` (public) |
| `GET /api/v1/integrations/services` | `GET /api/v1/n8n/services` |
| `GET /api/v1/integrations/providers` | `GET /api/v1/n8n/providers` |
| `GET /api/v1/integrations/availability` | `GET /api/v1/n8n/availability` |
| `POST /api/v1/integrations/appointments` | `POST /api/v1/n8n/appointments` |
| `POST /api/v1/integrations/inbound-messages` | `POST /api/v1/n8n/inbound-messages` |

### Changes Required

1. **Update URLs:** Change `/api/v1/integrations` to `/api/v1/n8n`
2. **Remove JWT Auth:** Delete `Authorization: Bearer <token>` header
3. **Add Headers:** Add `X-API-Key` and `X-Tenant-ID` headers
4. **Dynamic Tenant:** Tenant ID now comes from header/body, not JWT

---

## Support

For issues or questions:

1. Check logs: `docker compose -f docker-compose.prod.yml logs app`
2. Verify configuration: `cat app.env | grep N8N`
3. Test with cURL before using in n8n
4. Contact backend administrator with error details

---

## Changelog

### v1.0.0 (2026-07-10)

- Initial release of N8N integration routes
- Separate `/api/v1/n8n/*` namespace
- API Key authentication only (no JWT required)
- Dynamic multi-tenant support via headers
- Rate limiting: 500 req/min for N8N routes
- CORS configured for cross-origin requests
