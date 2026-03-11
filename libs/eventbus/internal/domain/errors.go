package domain

import "errors"

var (
	// ErrEventNotFound indica que el evento no existe
	ErrEventNotFound = errors.New("event not found")

	// ErrEventAlreadyProcessed indica que el evento ya fue procesado
	ErrEventAlreadyProcessed = errors.New("event already processed")

	// ErrInvalidEvent indica que el evento no es válido
	ErrInvalidEvent = errors.New("invalid event")

	// ErrConsumerNotFound indica que el consumidor no existe
	ErrConsumerNotFound = errors.New("consumer not found")

	// ErrMaxRetriesExceeded indica que se excedió el máximo de reintentos
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")

	// ErrEventStorageFailure indica un fallo en el almacenamiento
	ErrEventStorageFailure = errors.New("event storage failure")
)
