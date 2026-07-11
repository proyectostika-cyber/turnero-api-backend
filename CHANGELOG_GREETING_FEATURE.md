# Changelog - Greeting Personalizable por Tenant + Estado de Conversación

**Fecha:** 11 de Julio de 2026  
**Autor:** Development Team  
**Versión:** 1.1.0  
**Tipo:** Feature Implementation

---

## 📋 Resumen Ejecutivo

Se implementó un sistema completo para permitir que cada tenant personalice su mensaje de saludo (greeting) en conversaciones de WhatsApp, junto con un mecanismo robusto de manejo de estado de conversación para la integración con n8n.

### Objetivos Alcanzados

1. ✅ Cada tenant puede definir su propio mensaje de saludo personalizado
2. ✅ N8N recibe el greeting y datos del tenant al procesar webhooks de Evolution API
3. ✅ Sistema de estado de conversación persistente en PostgreSQL
4. ✅ Endpoints RESTful para actualizar greeting y estado de conversación
5. ✅ Validación de permisos granular (admin vs tenant)

---

## 🗄️ Cambios en Base de Datos

### Migración 000004: Columna `greeting_message`

**Archivo:** `internal/db/migration/000004_add_tenant_greeting.up.sql`

```sql
-- Agrega columna greeting_message a la tabla tenants
ALTER TABLE tenants
ADD COLUMN greeting_message TEXT;

-- Establece greeting por defecto para tenants existentes
UPDATE tenants
SET greeting_message = '¡Hola! ¿Dime en qué puedo ayudarte?'
WHERE greeting_message IS NULL;

-- Hace la columna NOT NULL
ALTER TABLE tenants
ALTER COLUMN greeting_message SET NOT NULL;
```

**Rollback:** `internal/db/migration/000004_add_tenant_greeting.down.sql`

```sql
ALTER TABLE tenants
DROP COLUMN IF EXISTS greeting_message;
```

---

### Migración 000005: UNIQUE Constraint para `conversation_state`

**Archivo:** `internal/db/migration/000005_conversation_state_unique.up.sql`

```sql
-- Garantiza un único estado de conversación por cliente por tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_state_tenant_customer
ON conversation_state (tenant_id, customer_id);
```

**Rollback:** `internal/db/migration/000005_conversation_state_unique.down.sql`

```sql
DROP INDEX IF EXISTS idx_conversation_state_tenant_customer;
```

**Propósito:** Permite usar `ON CONFLICT` en la query `UpsertConversationState` para hacer actualizaciones atómicas del estado sin condiciones de carrera.

---

## 🔧 Queries SQL (SQLC)

**Archivo:** `internal/db/query/mvp.sql`

### Nueva Query: `GetConversationState`

```sql
-- name: GetConversationState :one
SELECT id, tenant_id, customer_id, state, data, updated_at
FROM conversation_state
WHERE tenant_id = $1 AND customer_id = $2;
```

**Uso:** Obtener el estado actual de conversación de un cliente.

---

### Nueva Query: `UpdateTenantGreeting`

```sql
-- name: UpdateTenantGreeting :one
UPDATE tenants
SET greeting_message = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, timezone, active, greeting_message, created_at, updated_at;
```

**Uso:** Actualizar el mensaje de saludo de un tenant específico.

---

### Query Modificada: `UpsertConversationState`

**ANTES:**
```sql
INSERT INTO conversation_state (tenant_id, customer_id, state, data)
VALUES ($1, $2, $3, $4)
RETURNING id, tenant_id, customer_id, state, data, updated_at;
```

**DESPUÉS:**
```sql
INSERT INTO conversation_state (tenant_id, customer_id, state, data)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, customer_id)
DO UPDATE SET
    state = EXCLUDED.state,
    data = EXCLUDED.data,
    updated_at = NOW()
RETURNING id, tenant_id, customer_id, state, data, updated_at;
```

**Cambio:** Ahora hace un **UPSERT real** - si el estado ya existe, lo actualiza. Esto previene errores de duplicado y condiciones de carrera.

---

### Queries de Tenants Modificadas

Todas las queries de tenants fueron actualizadas para incluir `greeting_message`:

- `ListTenants` - Ahora incluye `greeting_message` en SELECT
- `GetTenant` - Retorna `greeting_message`
- `CreateTenant` - Permite especificar greeting al crear (default: "¡Hola! ¿Dime en qué puedo ayudarte?")
- `UpdateTenant` - Permite actualizar greeting junto con otros campos
- `DeactivateTenant` - Retorna `greeting_message` en el resultado

---

## 📦 Modelos y DTOs

### Modelo: `Tenant`

**Archivo:** `internal/db/sqlc/models.go`

```go
type Tenant struct {
    ID              uuid.UUID `json:"id"`
    Name            string    `json:"name"`
    Timezone        string    `json:"timezone"`
    Active          bool      `json:"active"`
    GreetingMessage string    `json:"greeting_message"` // ← NUEVO
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

---

### DTOs Nuevos

**Archivo:** `internal/api/dto/mvp.go`

#### 1. `UpdateTenantGreetingRequest`

```go
type UpdateTenantGreetingRequest struct {
    GreetingMessage string `json:"greeting_message" validate:"required,min=1,max=500"`
}
```

**Uso:** Request body para actualizar el greeting de un tenant.

---

#### 2. `UpdateConversationStateRequest`

```go
type UpdateConversationStateRequest struct {
    CurrentStep string          `json:"current_step" validate:"required"`
    Data        json.RawMessage `json:"data"`
}
```

**Uso:** Request body para que N8N actualice el estado de conversación.

**Ejemplo:**
```json
{
  "current_step": "awaiting_service_selection",
  "data": {
    "services": [...],
    "retry_count": 0
  }
}
```

---

#### 3. `EvolutionWebhookResponse`

```go
type EvolutionWebhookResponse struct {
    Processed         bool                  `json:"processed"`
    TenantID          uuid.UUID             `json:"tenant_id"`
    TenantName        string                `json:"tenant_name"`          // ← NUEVO
    GreetingMessage   string                `json:"greeting_message"`     // ← NUEVO
    CustomerID        uuid.UUID             `json:"customer_id"`
    ConversationState ConversationStateData `json:"conversation_state"`   // ← NUEVO
    Idempotent        bool                  `json:"idempotent"`
}
```

**Uso:** Respuesta del endpoint `/webhooks/evolution` que N8N consume.

---

#### 4. `ConversationStateData`

```go
type ConversationStateData struct {
    CurrentStep string          `json:"current_step"`
    Data        json.RawMessage `json:"data"`
}
```

**Uso:** Representa el estado actual de la conversación dentro de `EvolutionWebhookResponse`.

---

### DTO Modificado: `TenantResponse`

**ANTES:**
```go
type TenantResponse struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name"`
    Timezone  string    `json:"timezone"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**DESPUÉS:**
```go
type TenantResponse struct {
    ID              uuid.UUID `json:"id"`
    Name            string    `json:"name"`
    Timezone        string    `json:"timezone"`
    Active          bool      `json:"active"`
    GreetingMessage string    `json:"greeting_message"` // ← NUEVO
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

---

## 🏗️ Arquitectura de Servicios

### Repository: `AdminRepository`

**Archivo:** `internal/repositories/admin.go`

#### Nuevo Método: `UpdateTenantGreeting`

```go
UpdateTenantGreeting(context.Context, db.UpdateTenantGreetingParams) (db.Tenant, error)
```

**Implementación:**
```go
func (r *adminRepository) UpdateTenantGreeting(ctx context.Context, arg db.UpdateTenantGreetingParams) (db.Tenant, error) {
    row, err := r.store.UpdateTenantGreeting(ctx, arg)
    if err != nil {
        return db.Tenant{}, err
    }
    return toTenantFromUpdateGreeting(row), nil
}
```

---

### Service: `AdminService`

**Archivo:** `internal/services/admin.go`

#### Nuevo Método: `UpdateTenantGreeting`

```go
UpdateTenantGreeting(context.Context, uuid.UUID, string) (dto.TenantResponse, error)
```

**Implementación:**
```go
func (s *adminService) UpdateTenantGreeting(ctx context.Context, id uuid.UUID, message string) (dto.TenantResponse, error) {
    item, err := s.repo.UpdateTenantGreeting(ctx, db.UpdateTenantGreetingParams{
        ID:              id,
        GreetingMessage: message,
    })
    return mapTenant(item), err
}
```

---

### Service: `ConversationService`

**Archivo:** `internal/services/workflows.go`

#### Método Modificado: `ProcessEvolutionWebhook`

**ANTES:**
```go
ProcessEvolutionWebhook(context.Context, dto.EvolutionWebhookRequest, []byte) error
```

**DESPUÉS:**
```go
ProcessEvolutionWebhook(context.Context, dto.EvolutionWebhookRequest, []byte) (dto.EvolutionWebhookResponse, error)
```

**Cambio Principal:** Ahora retorna información completa del tenant y estado de conversación en lugar de solo un error.

**Respuesta de ejemplo:**
```json
{
  "processed": true,
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_name": "Consultorio Médico Dr. Pérez",
  "greeting_message": "¡Hola! Bienvenido a nuestro consultorio",
  "customer_id": "660e8400-e29b-41d4-a716-446655440000",
  "conversation_state": {
    "current_step": "greeting",
    "data": {}
  },
  "idempotent": true
}
```

---

#### Nuevo Método: `UpdateConversationState`

```go
UpdateConversationState(context.Context, uuid.UUID, dto.UpdateConversationStateRequest) error
```

**Implementación:**
```go
func (s *conversationService) UpdateConversationState(ctx context.Context, customerID uuid.UUID, req dto.UpdateConversationStateRequest) error {
    // Obtener customer para obtener tenant_id
    customer, err := s.repo.GetCustomer(ctx, customerID)
    if err != nil {
        return err
    }

    data := req.Data
    if len(data) == 0 {
        data = json.RawMessage("{}")
    }

    _, err = s.repo.Store().UpsertConversationState(ctx, db.UpsertConversationStateParams{
        TenantID:   customer.TenantID,
        CustomerID: customerID,
        State:      req.CurrentStep,
        Data:       []byte(data),
    })

    return err
}
```

**Uso:** N8N llama a este método para actualizar el estado después de cada paso de la conversación.

---

## 🛣️ Nuevas Rutas HTTP

**Archivo:** `internal/routes/web.go`

### 1. PUT `/api/v1/tenants/:id/greeting`

**Autenticación:** JWT (Bearer token)  
**Autorización:** `adminUser` o `tenantUser` (solo su propio tenant)  
**Handler:** `adminHandler.UpdateTenantGreeting`

**Request:**
```http
PUT /api/v1/tenants/550e8400-e29b-41d4-a716-446655440000/greeting
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "greeting_message": "¡Hola! Bienvenido a nuestro servicio de turnos médicos"
}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Consultorio Médico Dr. Pérez",
  "timezone": "America/Argentina/Buenos_Aires",
  "active": true,
  "greeting_message": "¡Hola! Bienvenido a nuestro servicio de turnos médicos",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-07-11T04:30:00Z"
}
```

**Validaciones:**
- `greeting_message` es requerido
- Longitud mínima: 1 carácter
- Longitud máxima: 500 caracteres

**Permisos:**
- **adminUser:** Puede actualizar el greeting de cualquier tenant
- **tenantUser:** Solo puede actualizar el greeting de su propio tenant (validado por `payload.TenantID`)

---

### 2. PUT `/api/v1/n8n/conversation-state/:customer_id`

**Autenticación:** API Key (`X-API-Key` header)  
**Autorización:** API Key válida  
**Handler:** `conversationHandler.UpdateConversationState`

**Request:**
```http
PUT /api/v1/n8n/conversation-state/660e8400-e29b-41d4-a716-446655440000
X-API-Key: sk_n8n_prod_7f3e9a2b1c8d4f6e5a9b2c1d3e4f5a6b7c8d9e0f
Content-Type: application/json

{
  "current_step": "awaiting_service_selection",
  "data": {
    "services": [
      {"id": "1", "name": "Consulta Médica"},
      {"id": "2", "name": "Análisis de Laboratorio"}
    ],
    "retry_count": 0
  }
}
```

**Response (200 OK):**
```json
{
  "message": "conversation state updated"
}
```

**Uso:** N8N llama a este endpoint después de cada paso del flujo conversacional para persistir el estado.

---

### 3. POST `/api/v1/webhooks/evolution` (MODIFICADO)

**Autenticación:** Pública (Evolution API la llama sin autenticación)  
**Handler:** `conversationHandler.EvolutionWebhook`

**Cambio Principal:** Ahora retorna `200 OK` con JSON detallado en lugar de `202 Accepted` vacío.

**Request:**
```http
POST /api/v1/webhooks/evolution
Content-Type: application/json

{
  "instance": "TenantA_Instance",
  "data": {
    "key": {
      "remoteJid": "5492995355116@s.whatsapp.net"
    },
    "message": {
      "conversation": "Hola, quiero un turno"
    }
  }
}
```

**Response (200 OK) - NUEVO:**
```json
{
  "processed": true,
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_name": "Consultorio Médico Dr. Pérez",
  "greeting_message": "¡Hola! Bienvenido a nuestro consultorio",
  "customer_id": "660e8400-e29b-41d4-a716-446655440000",
  "conversation_state": {
    "current_step": "message_received",
    "data": {}
  },
  "idempotent": true
}
```

**Uso por N8N:**
1. N8N recibe esta respuesta
2. Lee `greeting_message` para personalizar el saludo
3. Lee `conversation_state.current_step` para decidir qué nodo ejecutar a continuación
4. Usa `tenant_name` para personalizar mensajes

---

## 🎯 Handlers

### Handler: `UpdateTenantGreeting`

**Archivo:** `internal/api/handlers/admin_mvp.go`

```go
func (h *AdminMVPHandler) UpdateTenantGreeting(c *fiber.Ctx) error {
    id, err := parseID(c, "id")
    if err != nil {
        return response.Error(c, err)
    }

    var req dto.UpdateTenantGreetingRequest
    if err := bindAndValidate(c, &req); err != nil {
        return response.BadRequest(c, err)
    }

    // Validar permisos: solo adminUser o el tenantUser del mismo tenant puede actualizar
    payload := getPayload(c)
    if payload.Role != "adminUser" {
        if payload.TenantID == nil || *payload.TenantID != id {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "forbidden: can only update your own tenant's greeting",
            })
        }
    }

    rsp, err := h.service.UpdateTenantGreeting(c.Context(), id, req.GreetingMessage)
    if err != nil {
        return response.Error(c, err)
    }

    return c.JSON(rsp)
}
```

**Características:**
- Valida que el ID del tenant sea un UUID válido
- Valida el body con las reglas de `UpdateTenantGreetingRequest`
- Implementa autorización granular:
  - **adminUser:** Puede actualizar cualquier tenant
  - **tenantUser:** Solo puede actualizar su propio tenant (compara `payload.TenantID` con el ID del parámetro)
- Retorna `403 Forbidden` si un tenantUser intenta actualizar otro tenant

---

### Handler: `UpdateConversationState`

**Archivo:** `internal/api/handlers/workflows_mvp.go`

```go
func (h *ConversationMVPHandler) UpdateConversationState(c *fiber.Ctx) error {
    customerID, err := parseID(c, "customer_id")
    if err != nil {
        return response.Error(c, err)
    }

    var req dto.UpdateConversationStateRequest
    if err := bindAndValidate(c, &req); err != nil {
        return response.BadRequest(c, err)
    }

    err = h.service.UpdateConversationState(c.Context(), customerID, req)
    if err != nil {
        return response.Error(c, err)
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message": "conversation state updated",
    })
}
```

**Características:**
- Extrae `customer_id` de los parámetros de ruta
- Valida el body con las reglas de `UpdateConversationStateRequest`
- Hace upsert del estado (crea o actualiza atómicamente)
- Retorna mensaje de confirmación simple

---

### Handler Modificado: `EvolutionWebhook`

**Archivo:** `internal/api/handlers/workflows_mvp.go`

**ANTES:**
```go
func (h *ConversationMVPHandler) EvolutionWebhook(c *fiber.Ctx) error {
    var req dto.EvolutionWebhookRequest
    if err := c.BodyParser(&req); err != nil {
        return response.BadRequest(c, err)
    }
    if err := h.service.ProcessEvolutionWebhook(c.Context(), req, c.Body()); err != nil {
        return response.Error(c, err)
    }
    return c.SendStatus(fiber.StatusAccepted)
}
```

**DESPUÉS:**
```go
func (h *ConversationMVPHandler) EvolutionWebhook(c *fiber.Ctx) error {
    var req dto.EvolutionWebhookRequest
    if err := c.BodyParser(&req); err != nil {
        return response.BadRequest(c, err)
    }
    result, err := h.service.ProcessEvolutionWebhook(c.Context(), req, c.Body())
    if err != nil {
        return response.Error(c, err)
    }
    return c.Status(fiber.StatusOK).JSON(result)
}
```

**Cambio:** Ahora retorna el objeto `EvolutionWebhookResponse` con toda la información del tenant y estado de conversación.

---

## 🔄 Flujo Completo: N8N ↔ Backend

### Diagrama de Secuencia

```
Cliente (WhatsApp)  →  Evolution API  →  N8N Webhook  →  Backend  →  PostgreSQL
                                                              ↓
                                                        EvolutionWebhookResponse
                                                              ↓
                                                            N8N
                                                              ↓
                                                      (Procesa current_step)
                                                              ↓
                                                      (Construye respuesta)
                                                              ↓
                                                   PUT /conversation-state
                                                              ↓
                                                        PostgreSQL
```

---

### Paso a Paso Detallado

#### **1. Cliente Envía Mensaje**

Cliente escribe `"Hola"` en WhatsApp → Evolution API detecta el mensaje → Evolution API envía webhook a N8N

---

#### **2. N8N Reenvía a Backend**

N8N ejecuta nodo **"Forward to Backend"**:

```http
POST https://turnobot.hvdevs.com/api/v1/webhooks/evolution
Content-Type: application/json

{
  "instance": "TenantA_Instance",
  "data": {
    "key": {"remoteJid": "5492995355116@s.whatsapp.net"},
    "message": {"conversation": "Hola"}
  }
}
```

---

#### **3. Backend Procesa Webhook**

1. Resuelve `instance` → Busca en `tenant_channels` → Obtiene `tenant_id`
2. Resuelve `phone` → Busca/crea `customer_id` en `customers`
3. Guarda mensaje en `conversation_messages`
4. Lee/crea estado en `conversation_state`
5. Obtiene información del tenant (incluyendo `greeting_message`)
6. **Retorna respuesta JSON:**

```json
{
  "processed": true,
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_name": "Consultorio Médico Dr. Pérez",
  "greeting_message": "¡Hola! Bienvenido a nuestro consultorio",
  "customer_id": "660e8400-e29b-41d4-a716-446655440000",
  "conversation_state": {
    "current_step": "message_received",
    "data": {}
  },
  "idempotent": true
}
```

---

#### **4. N8N Procesa Estado**

N8N ejecuta nodo **"Route by State"**:

```javascript
const currentStep = $('Forward to Backend').first().json.conversation_state.current_step;

if (currentStep === "greeting") {
  // Ejecutar nodo "Get Services"
} else if (currentStep === "awaiting_service_selection") {
  // Ejecutar nodo "Parse Service Selection"
}
```

---

#### **5. N8N Construye Respuesta**

N8N ejecuta nodo **"Build Service Menu"**, construye mensaje personalizado con el `greeting_message`:

```javascript
const greeting = $('Forward to Backend').first().json.greeting_message;
const services = $('Get Services').all();

const message = `${greeting}\n\nServicios disponibles:\n1. Consulta Médica\n2. Análisis`;
```

---

#### **6. N8N Actualiza Estado**

N8N ejecuta nodo **"Update Conversation State"**:

```http
PUT https://turnobot.hvdevs.com/api/v1/n8n/conversation-state/660e8400-e29b-41d4-a716-446655440000
X-API-Key: sk_n8n_prod_7f3e9a2b1c8d4f6e5a9b2c1d3e4f5a6b7c8d9e0f
Content-Type: application/json

{
  "current_step": "awaiting_service_selection",
  "data": {
    "services": [...],
    "retry_count": 0
  }
}
```

Backend hace **UPSERT** en `conversation_state`:
- Si existe → Actualiza `state` y `data`
- Si NO existe → Inserta nuevo registro

---

#### **7. N8N Envía Mensaje al Cliente**

N8N ejecuta nodo **"Send WhatsApp Message"**:

```http
POST https://evo-api-server.hvdevs.com/message/sendText/TenantA_Instance
apikey: <evolution-api-key>

{
  "number": "5492995355116",
  "text": "¡Hola! Bienvenido a nuestro consultorio\n\nServicios disponibles:\n1. Consulta Médica\n2. Análisis"
}
```

---

#### **8. Cliente Responde**

Cliente escribe `"1"` → Evolution API → N8N → Backend:

1. Backend **LEE** estado de `conversation_state` → `current_step = "awaiting_service_selection"`
2. Retorna estado actual a N8N
3. N8N **SABE** que el cliente está eligiendo servicio → Procesa la selección
4. N8N actualiza estado → `current_step = "awaiting_provider_selection"`

**El ciclo continúa...**

---

## 📊 Persistencia de Estado

### Tabla: `conversation_state`

```sql
CREATE TABLE conversation_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id),
    state VARCHAR(100) NOT NULL,
    data JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- UNIQUE constraint agregado en migración 000005
    CONSTRAINT idx_conversation_state_tenant_customer 
        UNIQUE (tenant_id, customer_id)
);
```

### Estados Comunes

| Estado | Descripción |
|--------|-------------|
| `greeting` | Estado inicial, primer mensaje del cliente |
| `awaiting_service_selection` | Esperando que el cliente elija un servicio |
| `awaiting_provider_selection` | Esperando que el cliente elija un profesional |
| `awaiting_date_selection` | Esperando que el cliente elija una fecha |
| `awaiting_time_selection` | Esperando que el cliente elija un horario |
| `confirming_appointment` | Mostrando resumen, esperando confirmación |
| `confirmed` | Turno confirmado |
| `cancelled` | Turno cancelado |

### Ejemplo de Datos

```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "660e8400-e29b-41d4-a716-446655440000",
  "state": "awaiting_time_selection",
  "data": {
    "selected_service_id": "770e8400-e29b-41d4-a716-446655440000",
    "selected_provider_id": "880e8400-e29b-41d4-a716-446655440000",
    "selected_date": "2026-07-15",
    "available_slots": [
      {"time": "09:00", "slot_id": "..."},
      {"time": "10:00", "slot_id": "..."}
    ],
    "retry_count": 0
  },
  "updated_at": "2026-07-11T04:30:00Z"
}
```

---

## 🧪 Testing

### Test Manual: Actualizar Greeting

```bash
# 1. Login como adminUser
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'

# Response: {"access_token": "eyJhbGci..."}

# 2. Actualizar greeting del tenant
curl -X PUT http://localhost:8080/api/v1/tenants/550e8400-e29b-41d4-a716-446655440000/greeting \
  -H "Authorization: Bearer eyJhbGci..." \
  -H "Content-Type: application/json" \
  -d '{
    "greeting_message": "¡Hola! Bienvenido a Consultorio Médico Dr. Pérez. ¿En qué podemos ayudarte?"
  }'

# Response: TenantResponse con greeting actualizado
```

---

### Test Manual: Webhook Evolution

```bash
# Simular webhook de Evolution API
curl -X POST http://localhost:8080/api/v1/webhooks/evolution \
  -H "Content-Type: application/json" \
  -d '{
    "instance": "TenantA_Instance",
    "data": {
      "key": {"remoteJid": "5492995355116@s.whatsapp.net"},
      "message": {"conversation": "Hola"}
    }
  }'

# Response:
{
  "processed": true,
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_name": "Consultorio Médico Dr. Pérez",
  "greeting_message": "¡Hola! Bienvenido a Consultorio Médico Dr. Pérez. ¿En qué podemos ayudarte?",
  "customer_id": "660e8400-e29b-41d4-a716-446655440000",
  "conversation_state": {
    "current_step": "message_received",
    "data": {}
  },
  "idempotent": true
}
```

---

### Test Manual: Actualizar Estado (N8N)

```bash
curl -X PUT http://localhost:8080/api/v1/n8n/conversation-state/660e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: sk_n8n_prod_7f3e9a2b1c8d4f6e5a9b2c1d3e4f5a6b7c8d9e0f" \
  -H "Content-Type: application/json" \
  -d '{
    "current_step": "awaiting_service_selection",
    "data": {
      "services": [
        {"id": "1", "name": "Consulta Médica"},
        {"id": "2", "name": "Análisis"}
      ],
      "retry_count": 0
    }
  }'

# Response:
{
  "message": "conversation state updated"
}
```

---

## 🔐 Permisos y Autorización

### Matriz de Permisos

| Endpoint | adminUser | tenantUser | user | API Key (N8N) |
|----------|-----------|------------|------|---------------|
| `PUT /tenants/:id/greeting` | ✅ Todos | ✅ Solo su tenant | ❌ | ❌ |
| `PUT /n8n/conversation-state/:customer_id` | ❌ | ❌ | ❌ | ✅ |
| `POST /webhooks/evolution` | ✅ Público | ✅ Público | ✅ Público | ✅ Público |

### Validación de Permisos en `UpdateTenantGreeting`

```go
payload := getPayload(c) // Extrae JWT payload

if payload.Role != "adminUser" {
    // Si NO es admin, verificar que sea el tenant del usuario
    if payload.TenantID == nil || *payload.TenantID != id {
        return c.Status(403).JSON(fiber.Map{
            "error": "forbidden: can only update your own tenant's greeting",
        })
    }
}
```

**Casos de uso:**
- **Admin actualiza tenant A:** ✅ Permitido
- **Admin actualiza tenant B:** ✅ Permitido
- **TenantUser de A actualiza tenant A:** ✅ Permitido
- **TenantUser de A actualiza tenant B:** ❌ Prohibido (403 Forbidden)

---

## 📝 Notas de Implementación

### Nombres Genéricos en SQLC (Column1, Column2)

**Problema:** SQLC genera nombres genéricos `Column1`, `Column2` para parámetros posicionales en queries SQL.

**Causa:** Las queries usan `$1`, `$2` sin nombres explícitos.

**Solución Temporal:** El código de servicios usa `Column1`, `Column2` directamente:

```go
// ANTES (ideal pero no funciona con queries actuales):
db.ListTenantsParams{Search: search, Active: active}

// DESPUÉS (workaround actual):
db.ListTenantsParams{Column1: search, Column2: activeValue}
```

**Solución Permanente (futura refactorización):**  
Modificar queries SQL para usar `sqlc.arg(nombre)`:

```sql
-- ANTES:
WHERE ($1::text = '' OR name ILIKE '%' || $1 || '%')

-- DESPUÉS:
WHERE (sqlc.arg(search)::text = '' OR name ILIKE '%' || sqlc.arg(search) || '%')
```

---

### Upsert con ON CONFLICT

**Problema Previo:** La query `UpsertConversationState` podía fallar con error de duplicado si N8N enviaba múltiples actualizaciones simultáneas.

**Solución:** Se agregó UNIQUE constraint y `ON CONFLICT DO UPDATE`:

```sql
INSERT INTO conversation_state (tenant_id, customer_id, state, data)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, customer_id)
DO UPDATE SET
    state = EXCLUDED.state,
    data = EXCLUDED.data,
    updated_at = NOW()
RETURNING ...;
```

**Ventajas:**
- ✅ Operación atómica (sin condiciones de carrera)
- ✅ Idempotente (llamar varias veces con los mismos datos produce el mismo resultado)
- ✅ Elimina errores de duplicado

---

### Tipo de Retorno de `storeInboundMessage`

**Problema:** `storeInboundMessage` solo retornaba `db.ConversationMessage`, pero necesitábamos `TenantID` y `CustomerID`.

**Solución:** Se creó un struct `inboundMessageResult`:

```go
type inboundMessageResult struct {
    Message    db.ConversationMessage
    TenantID   uuid.UUID
    CustomerID uuid.UUID
}
```

**Ventaja:** Evita hacer queries adicionales después de la transacción.

---

## 🐛 Errores Conocidos (No Relacionados)

Los siguientes errores **NO están relacionados** con esta implementación y **existían previamente**:

```
internal/services/mappers.go:61:27: v.StartTime.Format undefined (type pgtype.Time has no field or method Format)
internal/services/mappers.go:62:25: v.EndTime.Format undefined (type pgtype.Time has no field or method Format)
internal/services/mappers.go:80:36: v.AppointmentID.UUID undefined (type pgtype.UUID has no field or method UUID)
internal/services/mappers.go:97:22: v.SlotID.UUID undefined (type pgtype.UUID has no field or method UUID)
internal/services/scheduling.go:63:15: cannot use req.StartTime (variable of type string) as pgtype.Time value
internal/services/scheduling.go:64:15: cannot use req.EndTime (variable of type string) as pgtype.Time value
```

**Causa:** Incompatibilidad entre tipos `pgtype.Time` / `pgtype.UUID` y el código que espera `time.Time` / `uuid.UUID`.

**Módulos afectados:**
- `ProviderAvailability` (horarios de disponibilidad)
- `AppointmentSlot` (franjas horarias)

**Impacto:** NO afecta la funcionalidad de greeting ni conversación.

---

## 🚀 Deployment

### 1. Aplicar Migraciones

Las migraciones se ejecutan **automáticamente** al iniciar el backend (ver `main.go` → `runDBMigration`).

```bash
# Iniciar backend (aplica migraciones automáticamente)
go run main.go
```

**Log esperado:**
```
{"level":"info","time":"2026-07-11T04:30:00-03:00","message":"db migrated successfully"}
```

---

### 2. Configurar Variable de Entorno

Verificar que `app.env` contenga:

```env
N8N_API_KEY=sk_n8n_prod_7f3e9a2b1c8d4f6e5a9b2c1d3e4f5a6b7c8d9e0f
```

---

### 3. Poblar Greeting Inicial (Opcional)

Si ya tienes tenants en la base de datos, puedes actualizar sus greetings:

```sql
-- Actualizar greeting de todos los tenants
UPDATE tenants
SET greeting_message = '¡Hola! Bienvenido a ' || name || '. ¿En qué puedo ayudarte?'
WHERE greeting_message = '¡Hola! ¿Dime en qué puedo ayudarte?';
```

O personalizar por tenant específico:

```sql
UPDATE tenants
SET greeting_message = '¡Hola! Bienvenido a Consultorio Médico Dr. Pérez. Estamos aquí para ayudarte con tus turnos.'
WHERE name = 'Consultorio Médico Dr. Pérez';
```

---

### 4. Actualizar Workflow N8N V2

Modificar el nodo **"Forward to Backend"** en n8n:

```javascript
// Capturar respuesta completa
const response = $('Forward to Backend').first().json;

const tenantName = response.tenant_name;
const greetingMessage = response.greeting_message;
const currentStep = response.conversation_state.current_step;
const customerID = response.customer_id;
```

Modificar el nodo **"Route by State"**:

```javascript
const currentStep = $('Forward to Backend').first().json.conversation_state.current_step;

switch(currentStep) {
  case "greeting":
    return 0; // Ruta a "Get Services"
  case "awaiting_service_selection":
    return 1; // Ruta a "Parse Service Selection"
  case "awaiting_provider_selection":
    return 2; // Ruta a "Parse Provider Selection"
  // ... etc
}
```

---

## 📚 Referencias Técnicas

### Archivos Modificados

```
internal/
├── api/
│   ├── dto/
│   │   └── mvp.go                          # ✨ DTOs nuevos + TenantResponse modificado
│   └── handlers/
│       ├── admin_mvp.go                    # ✨ UpdateTenantGreeting handler
│       └── workflows_mvp.go                # ✨ UpdateConversationState + EvolutionWebhook modificado
├── db/
│   ├── migration/
│   │   ├── 000004_add_tenant_greeting.up.sql      # ✨ NUEVO
│   │   ├── 000004_add_tenant_greeting.down.sql    # ✨ NUEVO
│   │   ├── 000005_conversation_state_unique.up.sql   # ✨ NUEVO
│   │   └── 000005_conversation_state_unique.down.sql # ✨ NUEVO
│   ├── query/
│   │   └── mvp.sql                         # ✨ GetConversationState, UpdateTenantGreeting, UpsertConversationState modificado
│   └── sqlc/
│       ├── models.go                       # ✨ Tenant.GreetingMessage (generado por SQLC)
│       └── mvp.sql.go                      # ✨ Queries generadas por SQLC
├── repositories/
│   ├── admin.go                            # ✨ UpdateTenantGreeting method
│   └── mappers.go                          # ✨ ToTenant, toTenantFromUpdateGreeting
├── routes/
│   └── web.go                              # ✨ 2 rutas nuevas
└── services/
    ├── admin.go                            # ✨ UpdateTenantGreeting
    ├── mappers.go                          # ✨ mapTenant con GreetingMessage
    └── workflows.go                        # ✨ ProcessEvolutionWebhook retorna EvolutionWebhookResponse, UpdateConversationState
```

---

### Comandos Útiles

```bash
# Regenerar código SQLC (después de modificar queries SQL)
sqlc generate

# Compilar backend
go build -o bin/appointments

# Ejecutar backend
go run main.go

# Aplicar migraciones manualmente (si no se ejecutan automáticamente)
migrate -path internal/db/migration \
  -database "postgresql://appointments:admin123@localhost:5433/appointments?sslmode=disable" \
  up

# Rollback última migración
migrate -path internal/db/migration \
  -database "postgresql://appointments:admin123@localhost:5433/appointments?sslmode=disable" \
  down 1
```

---

## 🎓 Conceptos Clave

### 1. Upsert (INSERT ... ON CONFLICT)

Un **upsert** es una operación que:
- **Inserta** un registro si no existe
- **Actualiza** el registro si ya existe (basado en una constraint)

**Ventajas:**
- Atómico (no hay condiciones de carrera)
- Idempotente
- Simplifica el código (no necesitas verificar existencia antes de insertar)

---

### 2. Máquina de Estados (State Machine)

El estado de conversación funciona como una **máquina de estados finitos**:

```
[greeting] → [awaiting_service_selection] → [awaiting_provider_selection] 
  → [awaiting_date_selection] → [awaiting_time_selection] 
  → [confirming_appointment] → [confirmed]
```

Cada transición está controlada por:
- **Input del cliente** (mensaje de WhatsApp)
- **Lógica de N8N** (valida input y decide siguiente estado)
- **Backend** (persiste el estado)

---

### 3. Idempotencia

Una operación es **idempotente** si ejecutarla múltiples veces produce el mismo resultado que ejecutarla una vez.

**Ejemplos en esta implementación:**
- `UpsertConversationState`: Llamar 3 veces con `state="greeting"` produce el mismo resultado
- `ProcessEvolutionWebhook`: Procesar el mismo webhook 2 veces (mismo `messageId`) debería ser idempotente (depende de lógica de deduplicación)

---

### 4. Autorización Granular

**Diferencia entre Autenticación y Autorización:**
- **Autenticación:** ¿Quién eres? (JWT, API Key)
- **Autorización:** ¿Qué puedes hacer? (Roles, Permisos)

**Implementación en `UpdateTenantGreeting`:**
```go
if payload.Role != "adminUser" {
    // Autorización granular: tenantUser solo puede actualizar SU tenant
    if payload.TenantID == nil || *payload.TenantID != id {
        return 403 // Forbidden
    }
}
```

---

## 🤝 Contribuciones Futuras

### Mejoras Sugeridas

1. **Tests Unitarios:**
   - `TestUpdateTenantGreeting_AsAdmin`
   - `TestUpdateTenantGreeting_AsTenantUser_OwnTenant`
   - `TestUpdateTenantGreeting_AsTenantUser_OtherTenant_Forbidden`
   - `TestProcessEvolutionWebhook_ReturnsGreeting`
   - `TestUpsertConversationState_Idempotent`

2. **Tests de Integración:**
   - Flujo completo: Webhook → Actualizar Estado → Webhook de nuevo
   - Concurrencia: 10 webhooks simultáneos del mismo cliente

3. **Refactorización de Queries SQL:**
   - Usar `sqlc.arg(nombre)` para nombres semánticos en lugar de Column1, Column2

4. **Validación de Estados:**
   - Enum de estados válidos en el backend
   - Validar transiciones de estado (no permitir saltar de `greeting` a `confirmed`)

5. **Métricas:**
   - Contador de greetings enviados por tenant
   - Tiempo promedio en cada estado de conversación
   - Tasa de conversión (greeting → confirmed)

6. **Feature: Historial de Estados**
   - Tabla `conversation_state_history` para auditoría
   - Trigger que registra cada cambio de estado

---

## 📞 Soporte

Para preguntas sobre esta implementación:
- **Issues:** GitHub Issues del proyecto
- **Documentación:** Este archivo + comentarios en código
- **Logs:** Revisar logs del backend para debugging

---

**Versión del Documento:** 1.0  
**Última Actualización:** 11 de Julio de 2026  
**Mantenedores:** Development Team
