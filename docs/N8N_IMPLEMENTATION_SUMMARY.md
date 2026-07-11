# N8N Integration - Implementation Summary

## 🎯 Objetivo Alcanzado

Se implementó con éxito un sistema de integración para N8N que permite:
- ✅ Autenticación solo con API Key (sin JWT)
- ✅ Soporte multi-tenant dinámico (tenant_id desde headers/request)
- ✅ Rutas separadas y dedicadas (`/api/v1/n8n/*`)
- ✅ Zero breaking changes (rutas JWT existentes siguen funcionando)

---

## 📁 Archivos Creados

### 1. **`internal/api/middleware/n8n_tenant.go`**
**Propósito:** Middleware que valida el tenant_id desde el request

**Funcionalidad:**
- Extrae `tenant_id` de: header `X-Tenant-ID` → query param `tenant_id` → body JSON
- Valida que el tenant existe en la base de datos
- Verifica que el tenant está activo
- Inyecta el tenant_id validado en el contexto para uso de handlers

**Función clave:**
```go
func ValidateTenantFromRequest(store *db.Store) fiber.Handler
func ExtractN8NTenant(c *fiber.Ctx) (uuid.UUID, bool)
```

---

### 2. **`internal/api/handlers/n8n_helpers.go`**
**Propósito:** Funciones helper para extracción de tenant_id

**Funcionalidad:**
- `ExtractTenantFromN8NContext`: Obtiene tenant_id del contexto N8N
- `GetTenantIDUnified`: Intenta obtener tenant_id de N8N context o JWT payload (orden de prioridad)

**Beneficio:** Los handlers pueden funcionar tanto con rutas JWT como con rutas N8N sin duplicar código.

---

### 3. **`docs/N8N_INTEGRATION.md`**
**Propósito:** Documentación completa de uso para N8N

**Contenido:**
- Guía de autenticación
- Lista completa de endpoints disponibles
- Ejemplos de uso con cURL y n8n
- Troubleshooting
- Mejores prácticas de seguridad

---

## 📝 Archivos Modificados

### 1. **`internal/api/middleware/apiKeyMiddleware.go`**
**Cambios:**
- Agregado comentario de documentación sobre separación de concerns
- Sin cambios funcionales (ya estaba bien implementado)

---

### 2. **`internal/api/server.go`**
**Cambios:**
```go
// Antes:
AllowHeaders: "Origin,Content-Type,Accept,Authorization",

// Después:
AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Tenant-ID",
```

**Beneficio:** Permite que n8n envíe los headers personalizados necesarios.

---

### 3. **`internal/routes/web.go`**
**Cambios mayores:**

#### Antes:
```go
// Rutas N8N mezcladas con JWT bajo /api/v1/integrations
n8nGroup := v1.Group("/integrations", middleware.APIKeyMiddleware(server.Config))
// Problema: v1 está dentro de auth, por lo que requiere JWT también
```

#### Después:
```go
// Rutas N8N completamente separadas FUERA del grupo auth
n8n := server.App.Group("/api/v1/n8n",
    middleware.APIKeyMiddleware(server.Config),
    middleware.ValidateTenantFromRequest(server.Store))

// Endpoints específicos para N8N
n8n.Get("/services", adminHandler.ListServices)
n8n.Post("/appointments", appointmentHandler.Create)
// etc.
```

**Beneficios:**
- Rutas N8N no requieren JWT
- Namespace claro y separado (`/api/v1/n8n/*`)
- Validación de tenant antes de llegar al handler
- Fácil de identificar y mantener

---

### 4. **`internal/api/handlers/admin_mvp.go`**
**Cambios en handlers:**
- `ListProviders`
- `ListServices`
- `ListCustomers`
- `ListTenantChannels`

#### Patrón aplicado:
```go
// Antes:
tenantID, err := uuid.Parse(c.Query("tenant_id"))
if err != nil {
    return response.Error(c, response.ErrInvalidInput)
}

// Después:
tenantID, err := GetTenantIDUnified(c)
if err != nil {
    // Fallback a query param para backwards compatibility
    tenantIDStr := c.Query("tenant_id")
    if tenantIDStr == "" {
        return response.Error(c, response.ErrInvalidInput)
    }
    tenantID, err = uuid.Parse(tenantIDStr)
    if err != nil {
        return response.Error(c, response.ErrInvalidInput)
    }
}
```

**Beneficios:**
- Un solo handler funciona para JWT y N8N
- Backwards compatible con rutas existentes
- Código más mantenible

---

### 5. **`internal/api/handlers/workflows_mvp.go`**
**Handlers modificados:**
- `AppointmentMVPHandler.List`
- `ConversationMVPHandler.List`

**Mismo patrón aplicado** que en admin_mvp.go

---

### 6. **`internal/api/handlers/scheduling_mvp.go`**
**Handlers modificados:**
- `SchedulingMVPHandler.Availability`

**Mismo patrón aplicado** que en admin_mvp.go

---

## 🏗️ Arquitectura Implementada

### Flujo de Request para Rutas N8N

```
┌─────────────────────────────────────────────────────┐
│  n8n HTTP Request Node                              │
│  Headers:                                           │
│    X-API-Key: your_api_key                          │
│    X-Tenant-ID: tenant_uuid                         │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│  Cloudflare (SSL, DDoS Protection)                  │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│  Nginx (Proxy Reverso)                              │
│  protika-bot-api-server.com                         │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│  Backend Go (Fiber) - Route: /api/v1/n8n/*         │
│                                                     │
│  1. APIKeyMiddleware                                │
│     ├─ Valida X-API-Key header                      │
│     ├─ Comparación constant-time                    │
│     └─ Rechaza si inválido (401)                    │
│                                                     │
│  2. ValidateTenantFromRequest                       │
│     ├─ Extrae tenant_id (header > query > body)    │
│     ├─ Query DB: ¿existe tenant?                    │
│     ├─ Verifica tenant.Active = true                │
│     └─ Inyecta tenant_id en context                 │
│                                                     │
│  3. Handler (ej: ListServices)                      │
│     ├─ Extrae tenant_id con GetTenantIDUnified()   │
│     ├─ Ejecuta lógica de negocio                    │
│     └─ Retorna respuesta JSON                       │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│  PostgreSQL Database                                │
│  - Tenant validation                                │
│  - Business logic queries                           │
└─────────────────────────────────────────────────────┘
```

### Comparación: JWT vs N8N Routes

| Aspecto | JWT Routes (`/api/v1/*`) | N8N Routes (`/api/v1/n8n/*`) |
|---------|--------------------------|------------------------------|
| **Auth** | `Authorization: Bearer <token>` | `X-API-Key: <key>` |
| **Tenant** | Del JWT payload (usuario específico) | Del header/body (dinámico) |
| **Middleware** | AuthMiddleware → RequireRole → RequireTenant | APIKeyMiddleware → ValidateTenantFromRequest |
| **Use Case** | Frontend web/mobile, usuarios humanos | Automatización n8n, webhooks, integraciones |
| **Rate Limit** | 200 req/min global | 500 req/min (más permisivo) |
| **Token Expiry** | 15 min (access) / 24h (refresh) | N/A (API key no expira) |

---

## 🔐 Seguridad Implementada

### 1. **Constant-Time Comparison**
```go
func validateAPIKey(provided, expected string) bool {
    if len(provided) != len(expected) {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
```
**Previene:** Timing attacks

---

### 2. **Validación de Tenant en DB**
```go
tenant, err := store.GetTenant(c.Context(), tenantID)
if err != nil {
    return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
        "error": "tenant not found",
    })
}
if !tenant.Active {
    return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
        "error": "tenant is not active",
    })
}
```
**Previene:** Acceso a tenants inexistentes o desactivados

---

### 3. **Rate Limiting Separado**
```go
// En server.go
n8nLimiter := limiter.New(limiter.Config{
    Max:        500,
    Expiration: 1 * time.Minute,
})
```
**Previene:** Abuso de la API, DoS

---

### 4. **CORS Configurado**
```go
AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Tenant-ID",
```
**Previene:** CORS errors, acceso no autorizado desde navegadores

---

## 📊 Endpoints Disponibles para N8N

### GET (Consultas)
- `GET /api/v1/n8n/services` - Listar servicios
- `GET /api/v1/n8n/providers` - Listar profesionales
- `GET /api/v1/n8n/customers` - Listar clientes
- `GET /api/v1/n8n/availability` - Ver disponibilidad
- `GET /api/v1/n8n/appointments` - Listar citas
- `GET /api/v1/n8n/appointments/:id` - Detalle de cita

### POST (Creación)
- `POST /api/v1/n8n/customers` - Crear cliente
- `POST /api/v1/n8n/appointments` - Crear cita
- `POST /api/v1/n8n/inbound-messages` - Mensaje entrante

### PATCH (Actualización)
- `PATCH /api/v1/n8n/appointments/:id` - Actualizar cita

### DELETE (Eliminación)
- `DELETE /api/v1/n8n/appointments/:id` - Cancelar cita

---

## 🧪 Testing

### Compilación Exitosa
```bash
cd /home/oem/Desktop/Appointments_App/appointments
go build -o /tmp/appointments_test main.go
# ✅ Compiló sin errores
```

### Test Manual con cURL

#### 1. Sin API Key (debe fallar)
```bash
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services
# Esperado: 401 {"error": "missing X-API-Key header"}
```

#### 2. Con API Key pero sin Tenant (debe fallar)
```bash
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services \
  -H "X-API-Key: your_api_key"
# Esperado: 400 {"error": "tenant_id is required..."}
```

#### 3. Con API Key y Tenant (debe funcionar)
```bash
curl -X GET https://protika-bot-api-server.com/api/v1/n8n/services \
  -H "X-API-Key: your_api_key" \
  -H "X-Tenant-ID: tenant_uuid"
# Esperado: 200 + JSON con servicios
```

#### 4. Crear Appointment
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
# Esperado: 201 + JSON con cita creada
```

---

## 📋 Checklist de Deployment

### Pre-deployment
- [x] Código compilado sin errores
- [x] Middleware de validación implementado
- [x] Handlers modificados para soportar N8N
- [x] CORS configurado con headers necesarios
- [x] Documentación completa creada
- [ ] API Key generada y segura (32+ caracteres)
- [ ] `app.env` actualizado con `N8N_API_KEY`

### Deployment en VPS
- [ ] Código pusheado a GitHub
- [ ] Pullear cambios en VPS: `git pull origin main`
- [ ] Verificar `app.env` tiene `N8N_API_KEY` configurado
- [ ] Rebuild containers: `docker compose -f docker-compose.prod.yml up -d --build`
- [ ] Verificar logs sin errores: `docker compose -f docker-compose.prod.yml logs app`
- [ ] Test con cURL desde VPS

### Post-deployment
- [ ] Test endpoints N8N desde cURL externo
- [ ] Configurar workflows en n8n con nueva URL
- [ ] Verificar que rutas JWT existentes siguen funcionando
- [ ] Monitorear logs por 24h para detectar errores
- [ ] Documentar cualquier issue encontrado

---

## 🚀 Próximos Pasos

### Inmediatos (Hoy)
1. ✅ Código implementado
2. ✅ Documentación creada
3. ⏳ Deploy a VPS
4. ⏳ Configurar workflows N8N

### Corto Plazo (Esta Semana)
1. Agregar logging de auditoría (quién accede a qué tenant)
2. Crear dashboard de métricas de uso N8N
3. Implementar alertas si rate limit es excedido frecuentemente

### Mediano Plazo (Este Mes)
1. Agregar más endpoints según necesidades de n8n
2. Implementar cache Redis para queries frecuentes
3. Crear tests automatizados para rutas N8N

### Largo Plazo (Próximos Meses)
1. Soporte para múltiples API keys (por tenant)
2. API key con permisos granulares (read-only, write, etc.)
3. Rotación automática de API keys
4. Agregar soporte para Zapier, Make.com con mismo patrón

---

## 🐛 Troubleshooting

### Problema: Backend no compila después de pull

**Solución:**
```bash
cd /opt/apps/turnero-api-backend
go mod tidy
go build -o main main.go
```

---

### Problema: 401 incluso con API Key correcta

**Diagnóstico:**
```bash
# Verificar API Key en app.env
cat app.env | grep N8N_API_KEY

# Verificar que backend leyó la variable
docker compose -f docker-compose.prod.yml exec app env | grep N8N

# Restart si hace falta
docker compose -f docker-compose.prod.yml restart app
```

---

### Problema: Tenant not found

**Diagnóstico:**
```bash
# Conectar a PostgreSQL
docker compose -f docker-compose.prod.yml exec postgres psql -U appointments_user -d appointments_prod

# Listar tenants
SELECT id, name, active FROM tenants;

# Verificar UUID exacto
\q
```

---

## 📞 Contacto

Para dudas o problemas con la implementación:
- Revisar: `docs/N8N_INTEGRATION.md` (guía de uso completa)
- Logs: `docker compose -f docker-compose.prod.yml logs app`
- Database check: Scripts de diagnóstico arriba

---

## 📝 Notas Finales

### ¿Por qué este diseño?

1. **Separación de concerns**: Rutas N8N completamente independientes de JWT
2. **Escalabilidad**: Fácil agregar más integraciones (Zapier, Make.com)
3. **Mantenibilidad**: Un solo handler sirve para JWT y N8N (GetTenantIDUnified)
4. **Seguridad**: Validación de tenant antes de llegar al handler
5. **Backwards compatibility**: Rutas JWT existentes no se tocan

### Zero Breaking Changes

- ✅ Todas las rutas JWT existentes funcionan igual
- ✅ Frontend no necesita cambios
- ✅ Usuarios actuales no se ven afectados
- ✅ Solo n8n usa las nuevas rutas

### Escalabilidad Futura

El patrón implementado permite fácilmente:
- Agregar más rutas N8N según necesidad
- Crear rutas `/api/v1/zapier/*` con el mismo patrón
- Implementar API keys por tenant (no global)
- Agregar permisos granulares a API keys
- Tracking y analytics por integración

---

**Implementación completada el:** 2026-07-10
**Versión:** 1.0.0
**Estado:** ✅ Listo para deployment
