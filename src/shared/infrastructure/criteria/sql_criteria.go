package criteria

import (
	"fmt"
	"strconv"
	"strings"

	domainCriteria "sales/src/shared/domain/criteria"
)

// SQLCriteriaConverter convierte un objeto Criteria en una consulta SQL
type SQLCriteriaConverter struct{}

// NewSQLCriteriaConverter crea una nueva instancia del conversor
func NewSQLCriteriaConverter() *SQLCriteriaConverter {
	return &SQLCriteriaConverter{}
}

// ToSelectSQL convierte un criteria a una consulta SQL SELECT completa con sus parámetros
func (s *SQLCriteriaConverter) ToSelectSQL(baseQuery string, criteria domainCriteria.Criteria) (string, []interface{}) {
	var parts []string
	var params []interface{}

	// Query base
	parts = append(parts, baseQuery)

	// Agregar WHERE clause si hay filtros
	if !criteria.Filters.IsEmpty() {
		whereClause, whereParams := s.buildWhereClause(criteria.Filters)
		parts = append(parts, whereClause)
		params = append(params, whereParams...)
	}

	// Agregar ORDER BY clause si hay ordenamiento
	if !criteria.Order.IsEmpty() {
		orderClause := s.buildOrderClause(criteria.Order)
		parts = append(parts, orderClause)
	}

	// Agregar LIMIT y OFFSET clause si hay paginación
	if criteria.Limit != nil && criteria.Offset != nil {
		limitClause := s.buildLimitClause(criteria.Limit, criteria.Offset)
		parts = append(parts, limitClause)
	}

	query := strings.Join(parts, " ")
	return query, params
}

// ToCountSQL convierte un criteria a una consulta SQL COUNT con sus parámetros
func (s *SQLCriteriaConverter) ToCountSQL(baseCountQuery string, criteria domainCriteria.Criteria) (string, []interface{}) {
	var parts []string
	var params []interface{}

	// Query base (generalmente "SELECT COUNT(*) FROM table")
	parts = append(parts, baseCountQuery)

	// Agregar WHERE clause si hay filtros
	if !criteria.Filters.IsEmpty() {
		whereClause, whereParams := s.buildWhereClause(criteria.Filters)
		parts = append(parts, whereClause)
		params = append(params, whereParams...)
	}

	// No necesitamos ORDER BY ni LIMIT para COUNT

	query := strings.Join(parts, " ")
	return query, params
}

// ToSQL convierte un criteria a una consulta SQL con sus parámetros (mantener para compatibilidad)
func (s *SQLCriteriaConverter) ToSQL(criteria domainCriteria.Criteria) (string, []interface{}) {
	var conditions []string
	var params []interface{}

	// Procesar los filtros
	for _, filter := range criteria.Filters.Items {
		condition, value := s.processFilter(filter)
		conditions = append(conditions, condition)
		if value != nil {
			params = append(params, value)
		}
	}

	// Construir la cláusula WHERE
	var whereClause string
	if len(conditions) > 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}

	// Construir la cláusula ORDER BY
	var orderByClause string
	if !criteria.Order.IsEmpty() {
		orderByClause = fmt.Sprintf("ORDER BY %s %s", criteria.Order.Field, string(criteria.Order.OrderType))
	}

	// Construir la cláusula LIMIT y OFFSET
	var limitOffsetClause string
	if criteria.Limit != nil && criteria.Offset != nil {
		limitOffsetClause = fmt.Sprintf("LIMIT %d OFFSET %d", *criteria.Limit, *criteria.Offset)
	}

	// Combinar las cláusulas
	clauses := []string{whereClause, orderByClause, limitOffsetClause}
	var filteredClauses []string
	for _, clause := range clauses {
		if clause != "" {
			filteredClauses = append(filteredClauses, clause)
		}
	}

	return strings.Join(filteredClauses, " "), params
}

// buildWhereClause construye la cláusula WHERE con sus parámetros
func (s *SQLCriteriaConverter) buildWhereClause(filters domainCriteria.Filters) (string, []interface{}) {
	var conditions []string
	var params []interface{}

	paramIndex := 1
	for _, filter := range filters.Items {
		condition, value := s.processFilterWithIndex(filter, paramIndex)
		conditions = append(conditions, condition)
		if value != nil {
			params = append(params, value)
			paramIndex++
		}
	}

	if len(conditions) > 0 {
		return fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND ")), params
	}

	return "", params
}

// buildOrderClause construye la cláusula ORDER BY
func (s *SQLCriteriaConverter) buildOrderClause(order domainCriteria.Order) string {
	return fmt.Sprintf("ORDER BY %s %s", order.Field, string(order.OrderType))
}

// buildLimitClause construye la cláusula LIMIT y OFFSET
func (s *SQLCriteriaConverter) buildLimitClause(limit, offset *int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", *limit, *offset)
}

// processFilterWithIndex convierte un filtro en una condición SQL con índice de parámetro
func (s *SQLCriteriaConverter) processFilterWithIndex(filter domainCriteria.Filter, paramIndex int) (string, interface{}) {
	var condition string
	placeholder := "$" + strconv.Itoa(paramIndex)

	switch filter.Operator {
	case domainCriteria.OpEqual, domainCriteria.OpNotEqual, domainCriteria.OpGreaterThan,
		domainCriteria.OpGreaterThanOrEqual, domainCriteria.OpLessThan, domainCriteria.OpLessThanOrEqual:
		condition = fmt.Sprintf("%s %s %s", filter.Field, filter.Operator, placeholder)
	case domainCriteria.OpLike:
		condition = fmt.Sprintf("%s LIKE %s", filter.Field, placeholder)
		// Asegurar que el valor sea compatible con LIKE
		if str, ok := filter.Value.(string); ok {
			if !strings.Contains(str, "%") {
				filter.Value = "%" + str + "%"
			}
		}
	case domainCriteria.OpIn:
		// Manejar arrays para cláusulas IN
		condition = fmt.Sprintf("%s IN (%s)", filter.Field, placeholder)
	case domainCriteria.OpIsNull:
		condition = fmt.Sprintf("%s IS NULL", filter.Field)
		return condition, nil
	case domainCriteria.OpIsNotNull:
		condition = fmt.Sprintf("%s IS NOT NULL", filter.Field)
		return condition, nil
	case domainCriteria.OpArrayContains:
		// PostgreSQL: para verificar si un array contiene un valor específico
		condition = fmt.Sprintf("%s @> ARRAY[%s]", filter.Field, placeholder)
	default:
		condition = fmt.Sprintf("%s = %s", filter.Field, placeholder)
	}

	return condition, filter.Value
}

// processFilter convierte un filtro en una condición SQL (mantener para compatibilidad)
func (s *SQLCriteriaConverter) processFilter(filter domainCriteria.Filter) (string, interface{}) {
	var condition string

	switch filter.Operator {
	case "=", "!=", ">", ">=", "<", "<=":
		condition = fmt.Sprintf("%s %s $?", filter.Field, filter.Operator)
	case "LIKE":
		condition = fmt.Sprintf("%s LIKE $?", filter.Field)
		// Asegurar que el valor sea compatible con LIKE
		if str, ok := filter.Value.(string); ok {
			if !strings.Contains(str, "%") {
				filter.Value = "%" + str + "%"
			}
		}
	case "IN":
		// Manejar arrays para cláusulas IN
		condition = fmt.Sprintf("%s IN ($?)", filter.Field)
	case "NULL":
		condition = fmt.Sprintf("%s IS NULL", filter.Field)
		return condition, nil
	case "NOT NULL":
		condition = fmt.Sprintf("%s IS NOT NULL", filter.Field)
		return condition, nil
	case "ARRAY_CONTAINS":
		// PostgreSQL: para verificar si un array contiene un valor específico
		condition = fmt.Sprintf("%s @> ARRAY[$?]", filter.Field)
	default:
		condition = fmt.Sprintf("%s = $?", filter.Field)
	}

	return condition, filter.Value
}
