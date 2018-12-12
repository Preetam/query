package query

import (
	"encoding/json"
	"errors"
)

var (
	ErrUnsupported = errors.New("query: unsupported query")
)

type Table interface {
	NewCursor() (Cursor, error)
}

type Cursor interface {
	Row() Row
	Next() bool
	Err() error
}

type Row interface {
	Fields() []string
	Get(field string) (interface{}, bool)
}

type Result struct {
	rows []resultRow
}

func (res *Result) Rows() []Row {
	rows := []Row{}
	for _, r := range res.rows {
		rows = append(rows, r)
	}
	return rows
}

type resultRow struct {
	values map[string]interface{}
}

func (r resultRow) Fields() []string {
	fields := []string{}
	for field := range r.values {
		fields = append(fields, field)
	}
	return fields
}

func (r resultRow) Get(field string) (interface{}, bool) {
	v, ok := r.values[field]
	return v, ok
}

func (r resultRow) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.values)
}

// Executor is a query executor.
type Executor struct {
	table Table
}

func NewExecutor(table Table) *Executor {
	return &Executor{
		table: table,
	}
}

// Execute executes a query and returns a set of rows for the result.
func (e *Executor) Execute(query *Query) (*Result, error) {
	// Get a cursor
	var cur Cursor
	var err error
	if len(query.Columns) == 1 && query.Columns[0].Name == "*" && len(query.GroupBy) == 0 {
		// SELECT * without GROUP BY
		cur, err = e.table.NewCursor()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrUnsupported
	}

	switch {
	case len(query.GroupBy) > 0, len(query.OrderBy) > 0:
		return nil, ErrUnsupported
	case len(query.Columns) > 0:
		for _, c := range query.Columns {
			if c.Aggregate != "" {
				return nil, ErrUnsupported
			}
		}
	}

	filters, err := buildFilters(query.Filters)
	if err != nil {
		return nil, err
	}

	resultRows := []resultRow{}
CursorLoop:
	for cur.Next() {
		for _, f := range filters {
			if !f.Filter(cur.Row()) {
				continue CursorLoop
			}
		}

		curRow := cur.Row()
		resRow := resultRow{
			values: map[string]interface{}{},
		}
		for _, field := range curRow.Fields() {
			v, _ := curRow.Get(field)
			resRow.values[field] = v
		}
		resultRows = append(resultRows, resRow)
		if query.Limit > 0 && len(resultRows) == query.Limit {
			break
		}
	}

	if cur.Err() != nil {
		return nil, cur.Err()
	}

	return &Result{rows: resultRows}, nil
}
