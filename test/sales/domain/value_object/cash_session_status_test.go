package value_object_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	vo "sales/src/sales/domain/value_object"
)

func TestCashSessionStatus_Transitions(t *testing.T) {
	cases := []struct {
		from vo.CashSessionStatus
		to   vo.CashSessionStatus
		ok   bool
	}{
		{vo.CashStatusOpen, vo.CashStatusClosed, true},
		{vo.CashStatusOpen, vo.CashStatusPendingReview, true},
		{vo.CashStatusOpen, vo.CashStatusClosing, true},
		{vo.CashStatusPendingReview, vo.CashStatusClosed, true},
		{vo.CashStatusClosing, vo.CashStatusClosed, true},
		// inválidas
		{vo.CashStatusClosed, vo.CashStatusOpen, false},
		{vo.CashStatusClosed, vo.CashStatusPendingReview, false},
		{vo.CashStatusPendingReview, vo.CashStatusOpen, false},
		{vo.CashStatusOpen, vo.CashStatusOpen, false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.ok, c.from.CanTransitionTo(c.to), "%s -> %s", c.from, c.to)
	}
}

func TestCashMovementType_Sign(t *testing.T) {
	assert.Equal(t, 1, vo.CashMovementIncome.Sign())
	assert.Equal(t, 1, vo.CashMovementCorrection.Sign())
	assert.Equal(t, -1, vo.CashMovementExpense.Sign())
	assert.Equal(t, -1, vo.CashMovementWithdrawal.Sign())

	assert.True(t, vo.CashMovementIncome.IsValid())
	assert.False(t, vo.CashMovementType("bogus").IsValid())
}
