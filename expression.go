package query

import (
	"strconv"
	"strings"
)

type expression struct {
	query          Query
	currentSection string
}

func (e *expression) AddColumn() {
	switch e.currentSection {
	case "columns":
		e.query.Columns = append(e.query.Columns, ColumnDesc{})
	case "group by":
		e.query.GroupBy = append(e.query.GroupBy, ColumnDesc{})
	case "order by":
		e.query.OrderBy = append(e.query.OrderBy, ColumnDesc{})
	}
}

func (e *expression) SetColumnName(name string) {
	switch e.currentSection {
	case "columns":
		e.query.Columns[len(e.query.Columns)-1].Name = name
	case "group by":
		e.query.GroupBy[len(e.query.GroupBy)-1].Name = name
	case "order by":
		e.query.OrderBy[len(e.query.OrderBy)-1].Name = name
	}
}

func (e *expression) SetColumnAggregate(aggregate string) {
	switch e.currentSection {
	case "columns":
		e.query.Columns[len(e.query.Columns)-1].Aggregate = aggregate
	case "group by":
		e.query.GroupBy[len(e.query.GroupBy)-1].Aggregate = aggregate
	case "order by":
		e.query.OrderBy[len(e.query.OrderBy)-1].Aggregate = aggregate
	}
}

func (e *expression) AddFilter() {
	e.query.Filters = append(e.query.Filters, FilterDesc{})
}

func (e *expression) SetFilterColumn(column string) {
	e.query.Filters[len(e.query.Filters)-1].Column = column
}

func (e *expression) SetFilterOperator(operator string) {
	e.query.Filters[len(e.query.Filters)-1].Operator = operator
}

func (e *expression) SetFilterValueFloat(value string) {
	f, _ := strconv.ParseFloat(value, 64)
	e.query.Filters[len(e.query.Filters)-1].Value = f
}

func (e *expression) SetFilterValueInteger(value string) {
	n, _ := strconv.ParseInt(value, 10, 64)
	e.query.Filters[len(e.query.Filters)-1].Value = int(n)
}

func (e *expression) SetFilterValueString(value string) {
	e.query.Filters[len(e.query.Filters)-1].Value = strings.Trim(value, `"`)
}

func (e *expression) SetDescending() {
	e.query.Descending = true
}

func (e *expression) SetLimit(num string) {
	e.query.Limit, _ = strconv.Atoi(num)
}

func Parse(query string) (*Query, error) {
	p := &parser{
		Buffer: query,
	}
	p.Init()
	err := p.Parse()
	if err != nil {
		return nil, err
	}
	p.Execute()
	return &p.query, nil
}
