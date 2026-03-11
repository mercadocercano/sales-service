# Arquitectura EventBus

## 🎯 Principios de Diseño

### 1. Hexagonal Architecture

```
┌──────────────────────────────────────────┐
│           Application Layer              │
│  (publish_event, process_event)          │
└──────────────┬───────────────────────────┘
               │
┌──────────────▼───────────────────────────┐
│           Domain Layer                   │
│  (DomainEvent, EventStore, EventHandler) │
└──────────────┬───────────────────────────┘
               │
┌──────────────▼───────────────────────────┐
│        Infrastructure Layer              │
│  (SQLEventStore, EventWorker, Config)    │
└──────────────────────────────────────────┘
```

### 2. Domain-Driven Design

- **Aggregate**: Event (inmutable)
- **Value Objects**: ConsumerStatus
- **Repository**: EventStore (port)
- **Domain Services**: N/A (infraestructura pura)

### 3. SOLID Principles

- **S**: Cada caso de uso tiene una responsabilidad única
- **O**: Nuevos adaptadores sin modificar interfaces
- **L**: SQLEventStore sustituible por RedisEventStore
- **I**: Interfaces pequeñas y específicas
- **D**: Dependencias apuntan a abstracciones (ports)

## 🔄 Flujo de Procesamiento

### Publicación de Evento

```
Publisher → PublishEventUseCase → EventStore.Save() → PostgreSQL
```

### Procesamiento de Evento

```
EventWorker → ProcessEventUseCase → EventStore.GetUnprocessed()
           ↓
       EventHandler.Handle()
           ↓
   EventStore.MarkProcessed() / MarkFailed()
```

## 🗃️ Diseño de Base de Datos

### Decisiones Arquitectónicas

1. **Sin Foreign Keys**: Permite escalar horizontalmente sin problemas de locks
2. **Sin UNIQUE Constraints**: Idempotencia controlada en código
3. **Sin Triggers**: Toda la lógica en Go para testing y debugging
4. **JSONB Payload**: Flexibilidad para evolucionar esquemas
5. **Índices Estratégicos**: Performance en queries frecuentes

### Queries Críticas

#### GetUnprocessed

```sql
SELECT eb.*
FROM event_bus eb
WHERE NOT EXISTS (
    SELECT 1 FROM event_consumers ec
    WHERE ec.event_id = eb.id
    AND ec.consumer_name = ?
    AND ec.status = 'processed'
)
AND (
    SELECT COALESCE(MAX(retry_count), 0)
    FROM event_consumers
    WHERE event_id = eb.id AND consumer_name = ?
) < 3
ORDER BY eb.occurred_at ASC
LIMIT ?
```

**Optimizaciones**:
- Índice en `(consumer_name, status)`
- Índice en `(consumer_name, retry_count)`
- Límite de batch para evitar memory issues

## 🔁 Retry Strategy

### Backoff Exponencial

```
Retry 1: 5 segundos
Retry 2: 10 segundos (2x)
Retry 3: 20 segundos (2x)
Max: 3 reintentos → status = 'failed'
```

### Estados de Consumidor

```
pending → processing → processed ✅
       ↓
    retrying (retry_count < 3)
       ↓
    failed (retry_count >= 3) ❌
```

## 🔒 Garantías y Trade-offs

### Garantías

- ✅ **Durabilidad**: Eventos persistidos antes de confirmar
- ✅ **At-least-once**: Un evento puede procesarse múltiples veces
- ✅ **Orden por agregado**: Eventos del mismo agregado en orden
- ✅ **No pérdida de datos**: Append-only sin deletes

### Trade-offs

- ❌ **Exactly-once**: No garantizado (requiere idempotencia)
- ❌ **Latencia baja**: Polling cada 5s (configurable)
- ❌ **Orden global**: Solo garantizado por agregado

## 🚀 Escalabilidad

### Horizontal Scaling

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  Worker 1    │   │  Worker 2    │   │  Worker 3    │
│  (Ledger)    │   │  (Stock)     │   │  (Fiscal)    │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       └──────────────────┴──────────────────┘
                          │
                   ┌──────▼───────┐
                   │  PostgreSQL  │
                   │  (EventBus)  │
                   └──────────────┘
```

**Estrategias**:
- Un worker por consumidor
- Múltiples instancias del mismo consumidor (requiere locking)
- Particionamiento por aggregate_type

### Vertical Scaling

- Aumentar `WORKER_BATCH_SIZE`
- Reducir `WORKER_POLL_INTERVAL`
- Agregar índices adicionales
- Connection pooling

## 🔄 Migración a Otros Backends

### Redis

```go
type RedisEventStore struct {
    client *redis.Client
}

func (s *RedisEventStore) Save(ctx context.Context, event DomainEvent) error {
    // Stream: XADD events:sale:* ...
}

func (s *RedisEventStore) GetUnprocessed(...) ([]DomainEvent, error) {
    // Consumer Group: XREADGROUP ...
}
```

### Kafka

```go
type KafkaEventStore struct {
    producer *kafka.Producer
    consumer *kafka.Consumer
}

func (s *KafkaEventStore) Save(ctx context.Context, event DomainEvent) error {
    // Topic: events.sale.created
    // Partition by aggregate_id
}
```

### EventStoreDB

```go
type EventStoreDBStore struct {
    client *esdb.Client
}

func (s *EventStoreDBStore) Save(ctx context.Context, event DomainEvent) error {
    // Stream: sale-{aggregate_id}
}
```

## 🧪 Testing Strategy

### Unit Tests

- Domain: Entidades y value objects
- Application: Casos de uso (con mocks)
- Infrastructure: Adapters (con testcontainers)

### Integration Tests

```go
func TestPublishAndConsume(t *testing.T) {
    // 1. Setup PostgreSQL testcontainer
    // 2. Publicar evento
    // 3. Consumir evento
    // 4. Verificar procesamiento
}
```

### E2E Tests

```bash
# 1. Publicar evento desde sales-service
# 2. Verificar procesamiento en ledger-service
# 3. Verificar procesamiento en stock-service
# 4. Verificar estado en event_consumers
```

## 📊 Observabilidad

### Logs Estructurados

```json
{
  "timestamp": "2025-02-19T10:30:00Z",
  "level": "INFO",
  "message": "Event processed",
  "event_id": "abc-123",
  "consumer": "ledger-service",
  "event_type": "sale.created",
  "duration_ms": 150
}
```

### Métricas (futuro)

```
eventbus_events_published_total{aggregate_type, event_type}
eventbus_events_processed_total{consumer, event_type, status}
eventbus_processing_duration_seconds{consumer, event_type}
eventbus_retry_count{consumer, event_type}
```

## 🔐 Security

- **No autenticación**: EventBus es interno, no expone HTTP
- **Validación de payload**: Responsabilidad de publishers
- **SQL Injection**: Protegido por prepared statements
- **Secrets**: Variables de entorno, nunca en código

## 🎯 Roadmap

### Fase 1 - MVP ✅
- [x] Interfaces de dominio
- [x] SQL implementation
- [x] Worker genérico
- [x] Retry básico
- [x] Ejemplos de uso

### Fase 2 - Production Ready
- [ ] Dead Letter Queue
- [ ] Métricas Prometheus
- [ ] Health checks
- [ ] Graceful shutdown mejorado
- [ ] Circuit breaker

### Fase 3 - Advanced
- [ ] Redis adapter
- [ ] Kafka adapter
- [ ] Event Sourcing completo
- [ ] Snapshots
- [ ] CQRS integration

---

**Arquitectura aprobada para HITO 1** ✅
