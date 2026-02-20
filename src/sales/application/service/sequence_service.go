package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SequenceService gestiona numeración secuencial con optimistic locking
type SequenceService struct {
	db *sql.DB
}

// NewSequenceService crea una nueva instancia
func NewSequenceService(db *sql.DB) *SequenceService {
	return &SequenceService{
		db: db,
	}
}

// NextNumber obtiene el siguiente número de secuencia para un tipo de documento
// Implementa optimistic locking con retry automático
func (s *SequenceService) NextNumber(ctx context.Context, tenantID, documentType string) (int, error) {
	const maxRetries = 5
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		number, err := s.tryGetNextNumber(ctx, tenantID, documentType)
		
		if err == nil {
			// Éxito
			return number, nil
		}
		
		// Si no es error de concurrencia, fallar inmediatamente
		if err != ErrConcurrentUpdate {
			return 0, err
		}
		
		// Retry con backoff exponencial
		if attempt < maxRetries-1 {
			backoff := time.Duration(attempt+1) * 10 * time.Millisecond
			log.Printf("Optimistic locking conflict on attempt %d, retrying in %v...", attempt+1, backoff)
			time.Sleep(backoff)
		}
	}
	
	return 0, fmt.Errorf("failed to get next number after %d retries", maxRetries)
}

// tryGetNextNumber intenta obtener el siguiente número (single attempt)
func (s *SequenceService) tryGetNextNumber(ctx context.Context, tenantID, documentType string) (int, error) {
	// 1. Leer secuencia actual con version
	querySel := `
		SELECT current_number, version
		FROM document_sequences
		WHERE tenant_id = $1 AND document_type = $2
	`
	
	var currentNumber int
	var version int
	
	err := s.db.QueryRowContext(ctx, querySel, tenantID, documentType).Scan(&currentNumber, &version)
	
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("sequence not found for tenant %s, document_type %s", tenantID, documentType)
	}
	
	if err != nil {
		return 0, fmt.Errorf("error reading sequence: %w", err)
	}
	
	// 2. Calcular nuevo número
	newNumber := currentNumber + 1
	newVersion := version + 1
	
	// 3. Actualizar con optimistic locking (WHERE version = oldVersion)
	queryUpd := `
		UPDATE document_sequences
		SET current_number = $1, version = $2, updated_at = $3
		WHERE tenant_id = $4 AND document_type = $5 AND version = $6
	`
	
	result, err := s.db.ExecContext(
		ctx,
		queryUpd,
		newNumber,
		newVersion,
		time.Now(),
		tenantID,
		documentType,
		version, // WHERE version = oldVersion
	)
	
	if err != nil {
		return 0, fmt.Errorf("error updating sequence: %w", err)
	}
	
	// 4. Verificar si se actualizó (optimistic locking check)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error checking rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		// Otro proceso actualizó la secuencia (version cambió)
		return 0, ErrConcurrentUpdate
	}
	
	log.Printf("✅ Sequence assigned: tenant=%s, type=%s, number=%d, version=%d", tenantID, documentType, newNumber, newVersion)
	
	return newNumber, nil
}

// ErrConcurrentUpdate indica que hubo una actualización concurrente
var ErrConcurrentUpdate = fmt.Errorf("concurrent update detected")
