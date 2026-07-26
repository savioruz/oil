// Package dto provides data transfer objects for the application.
package dto

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
)

const (
	// FilterOperatorEq is used to indicate that the filter should check for equality between the field and the value (e.g., field = value)
	FilterOperatorEq = "eq"
	// FilterOperatorLike is used to indicate that the filter should check for a pattern match using the SQL LIKE operator (e.g., field LIKE '%value%')
	FilterOperatorLike = "like"
	// FilterOperatorIn is used to indicate that the filter should check if the field's value is within a specified set of values using the SQL IN operator (e.g., field IN (value1, value2, ...))
	FilterOperatorIn = "in"
	// FilterOperatorNotEq is used to indicate that the filter should check for inequality between the field and the value (e.g., field != value)
	FilterOperatorNotEq = "not_eq"
	// FilterOperatorLessEq is used to indicate that the filter should check if the field's value is less than or equal to the specified value (e.g., field <= value)
	FilterOperatorLessEq = "less_eq"
	// FilterOperatorLessThan is used to indicate that the filter should check if the field's value is less than the specified value (e.g., field < value)
	FilterOperatorLessThan = "less_than"
	// FilterOperatorGreaterEq is used to indicate that the filter should check if the field's value is greater than or equal to the specified value (e.g., field >= value)
	FilterOperatorGreaterEq = "greater_eq"
	// FilterOperatorGreaterThan is used to indicate that the filter should check if the field's value is greater than the specified value (e.g., field > value)
	FilterOperatorGreaterThan = "greater_than"
	// FilterPlainQuery is used to indicate that the filter value is a raw SQL query that should be included directly in the WHERE clause without any modification or parameterization (e.g., (field1 = value1 AND field2 > value2))
	FilterPlainQuery = "plan"
	// FilterIsNotNull is used to indicate that the filter should check if the field's value is not null (e.g., field IS NOT NULL)
	FilterIsNotNull = "is_not_null"
	// FilterIsNull is used to indicate that the filter should check if the field's value is null (e.g., field IS NULL)
	FilterIsNull = "is_null"
)

const (
	// FilterGroupOperatorAnd is used to indicate that filters in a FilterGroup should be combined with a logical AND
	FilterGroupOperatorAnd = "AND"
	// FilterGroupOperatorOr is used to indicate that filters in a FilterGroup should be combined with a logical OR
	FilterGroupOperatorOr = "OR"
)

// Filter represents a single filter condition for querying data.
// It includes the field to filter on, the operator to use, and the value to compare against.
// The GetWhereClause method generates the corresponding SQL WHERE clause and a map of named parameters for query execution.
type Filter struct {
	ArgName  string
	Field    string
	Value    any
	Operator string `validate:"required,oneof=eq like in not_eq less_eq greater_eq"`
	Table    string
}

// GetWhereClause generates the SQL WHERE clause for the filter based on its operator and value,
// and returns the clause along with a map of named parameters for query execution.
func (f Filter) GetWhereClause() (string, map[string]any) {
	args := map[string]any{}

	column := f.Field
	if f.Table != "" {
		column = f.Table + "." + f.Field
	}

	argName := f.ArgName
	if argName == "" {
		argName = f.Field
	}

	switch f.Operator {
	case FilterOperatorEq:
		args[argName] = f.Value

		return column + " = :" + argName, args
	case FilterOperatorLike:
		args[argName] = "%" + f.Value.(string) + "%"

		return "LOWER(" + column + ") LIKE LOWER(:" + argName + ") ", args
	case FilterOperatorIn:
		named := inArgs(argName, f.Value, args)

		return column + " IN (" + strings.Join(named, ", ") + ") ", args
	case FilterOperatorNotEq:
		args[argName] = f.Value

		return column + " != :" + argName, args
	case FilterOperatorLessEq:
		args[argName] = f.Value

		return column + " <= :" + argName, args
	case FilterOperatorGreaterEq:
		args[argName] = f.Value

		return column + " >= :" + argName, args
	case FilterPlainQuery:
		query, _ := f.Value.(string)

		return "(" + query + ")", args
	case FilterIsNotNull:
		return column + " IS NOT NULL", args
	case FilterIsNull:
		return column + " IS NULL", args
	default:
		return "", args
	}
}

// FilterGroup represents a group of filters combined with a logical operator (AND/OR). It can contain both individual filters and nested filter groups, allowing for complex query conditions.
type FilterGroup struct {
	Filters  []any
	Operator string
}

// GetWhereClause generates the combined WHERE clause for the filter group, including all nested filters and groups, and returns the clause along with a map of named parameters for query execution.
func (f FilterGroup) GetWhereClause() (string, map[string]any) {
	args := make(map[string]any, len(f.Filters))
	whereClause := make([]string, 0, len(f.Filters))

	for _, filter := range f.Filters {
		switch fill := filter.(type) {
		case Filter:
			where, arg := fill.GetWhereClause()
			whereClause = append(whereClause, where)

			maps.Copy(args, arg)
		case FilterGroup:
			where, arg := fill.GetWhereClause()
			whereClause = append(whereClause, where)

			maps.Copy(args, arg)
		}
	}

	if len(whereClause) == 0 {
		return "", args
	}

	return "(" + strings.Join(whereClause, " "+f.Operator+" ") + ")", args
}

// inArgs populates 'args' from a slice value and returns the placeholder names.
// It uses a type-switch over common slice types instead of reflect.
func inArgs(argName string, value any, args map[string]any) []string {
	switch v := value.(type) {
	case []string:
		named := make([]string, len(v))
		for i, s := range v {
			var b strings.Builder
			b.WriteByte(':')
			b.WriteString(argName)
			b.WriteByte('_')
			b.WriteString(strconv.Itoa(i))
			named[i] = b.String()
			args[named[i][1:]] = s
		}

		return named
	case []int:
		named := make([]string, len(v))
		for i, n := range v {
			var b strings.Builder
			b.WriteByte(':')
			b.WriteString(argName)
			b.WriteByte('_')
			b.WriteString(strconv.Itoa(i))
			named[i] = b.String()
			args[named[i][1:]] = n
		}

		return named
	case []int64:
		named := make([]string, len(v))
		for i, n := range v {
			var b strings.Builder
			b.WriteByte(':')
			b.WriteString(argName)
			b.WriteByte('_')
			b.WriteString(strconv.Itoa(i))
			named[i] = b.String()
			args[named[i][1:]] = n
		}

		return named
	case []any:
		named := make([]string, len(v))
		for i, a := range v {
			var b strings.Builder
			b.WriteByte(':')
			b.WriteString(argName)
			b.WriteByte('_')
			b.WriteString(strconv.Itoa(i))
			named[i] = b.String()
			args[named[i][1:]] = a
		}

		return named
	default:
		return []string{fmt.Sprint(value)}
	}
}
