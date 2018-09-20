package query

import "encoding/json"

// Query describes a query.
type Query struct {
	Columns    []ColumnDesc `json:"columns,omitempty"`
	GroupBy    []ColumnDesc `json:"group_by,omitempty"`
	Filters    []FilterDesc `json:"filters,omitempty"`
	OrderBy    []ColumnDesc `json:"order_by,omitempty"`
	Descending bool         `json:"descending"`
	Limit      int          `json:"limit,omitempty"`
}

// ColumnDesc describes a column.
type ColumnDesc struct {
	Name      string `json:"name"`
	Aggregate string `json:"aggregate,omitempty"`
}

// FilterDesc represents a filter expression.
type FilterDesc struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

func (q Query) String() string {
	b, _ := json.Marshal(q)
	return string(b)
}
