package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sales/src/sales/domain/entity"
	"sales/src/sales/domain/value_object"
)

func openSession(t *testing.T, openedBy uuid.UUID) *entity.CashRegisterSession {
	t.Helper()
	s, err := entity.NewCashRegisterSession(
		uuid.New(), "POS-1", openedBy,
		decimal.NewFromInt(1000), // opening
		decimal.NewFromInt(100),  // review threshold
	)
	require.NoError(t, err)
	require.Equal(t, value_object.CashStatusOpen, s.Status)
	return s
}

func TestNewCashRegisterSession_Validations(t *testing.T) {
	_, err := entity.NewCashRegisterSession(uuid.New(), "", uuid.New(), decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, entity.ErrPointOfSaleRequired)

	_, err = entity.NewCashRegisterSession(uuid.New(), "POS-1", uuid.Nil, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, entity.ErrOpenedByRequired)

	_, err = entity.NewCashRegisterSession(uuid.New(), "POS-1", uuid.New(), decimal.NewFromInt(-1), decimal.Zero)
	assert.ErrorIs(t, err, entity.ErrOpeningAmountNegative)
}

func TestClose_WithinThreshold_Closed(t *testing.T) {
	s := openSession(t, uuid.New())
	closer := uuid.New()

	// counted 1080, expected 1000 -> diff +80, |80| <= 100 -> closed
	err := s.Close(closer, decimal.NewFromInt(1080), decimal.NewFromInt(1000))
	require.NoError(t, err)

	assert.Equal(t, value_object.CashStatusClosed, s.Status)
	require.NotNil(t, s.Difference)
	assert.True(t, s.Difference.Equal(decimal.NewFromInt(80)), "difference debe ser counted-expected")
	require.NotNil(t, s.ClosedBy)
	assert.Equal(t, closer, *s.ClosedBy)
}

func TestClose_ExceedsThreshold_PendingReview(t *testing.T) {
	s := openSession(t, uuid.New())

	// counted 800, expected 1000 -> diff -200, |200| > 100 -> pending_review
	err := s.Close(uuid.New(), decimal.NewFromInt(800), decimal.NewFromInt(1000))
	require.NoError(t, err)

	assert.Equal(t, value_object.CashStatusPendingReview, s.Status)
	require.NotNil(t, s.Difference)
	assert.True(t, s.Difference.Equal(decimal.NewFromInt(-200)))
}

func TestClose_NotOpen_Fails(t *testing.T) {
	s := openSession(t, uuid.New())
	require.NoError(t, s.Close(uuid.New(), decimal.NewFromInt(1000), decimal.NewFromInt(1000)))

	// segundo cierre sobre una caja ya cerrada
	err := s.Close(uuid.New(), decimal.NewFromInt(1000), decimal.NewFromInt(1000))
	assert.ErrorIs(t, err, entity.ErrCashSessionNotOpen)
}

func TestClose_NegativeCounted_Fails(t *testing.T) {
	s := openSession(t, uuid.New())
	err := s.Close(uuid.New(), decimal.NewFromInt(-1), decimal.NewFromInt(1000))
	assert.ErrorIs(t, err, entity.ErrCountedAmountNegative)
}

// Separación de funciones (ADR-003 gate #2): el aprobador no puede ser ni el que abrió
// ni el que cerró la caja.
func TestApproveReview_SeparationOfDuties(t *testing.T) {
	operator := uuid.New()
	closer := uuid.New()
	supervisor := uuid.New()

	pending := func() *entity.CashRegisterSession {
		s := openSession(t, operator)
		require.NoError(t, s.Close(closer, decimal.NewFromInt(500), decimal.NewFromInt(1000))) // diff -500 -> pending
		require.Equal(t, value_object.CashStatusPendingReview, s.Status)
		return s
	}

	// Aprobar siendo el cajero que abrió -> rechazado
	s := pending()
	assert.ErrorIs(t, s.ApproveReview(operator), entity.ErrApproverIsOperator)
	assert.Equal(t, value_object.CashStatusPendingReview, s.Status)

	// Aprobar siendo quien cerró -> rechazado
	s = pending()
	assert.ErrorIs(t, s.ApproveReview(closer), entity.ErrApproverIsOperator)

	// Aprobar siendo un tercero (supervisor) -> OK, queda closed
	s = pending()
	require.NoError(t, s.ApproveReview(supervisor))
	assert.Equal(t, value_object.CashStatusClosed, s.Status)
	require.NotNil(t, s.ApprovedBy)
	assert.Equal(t, supervisor, *s.ApprovedBy)
}

func TestApproveReview_NotInReview_Fails(t *testing.T) {
	s := openSession(t, uuid.New())
	require.NoError(t, s.Close(uuid.New(), decimal.NewFromInt(1000), decimal.NewFromInt(1000))) // closed OK
	err := s.ApproveReview(uuid.New())
	assert.ErrorIs(t, err, entity.ErrCashSessionNotInReview)
}
