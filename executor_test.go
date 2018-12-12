package query

import "testing"

var testData = []map[string]interface{}{
	{"id": 1, "a": 1, "b": 2},
	{"id": 2, "a": 1, "b": 2},
	{"id": 3, "a": 1, "b": 2},
	{"id": 4, "a": 1, "b": 2},
}

type testDataCursor struct {
	idx  int
	data []map[string]interface{}
}

func (c *testDataCursor) Err() error {
	return nil
}

func (c *testDataCursor) Next() bool {
	c.idx++
	if c.idx < len(c.data) {
		return true
	}
	return false
}

func (c *testDataCursor) Row() Row {
	return resultRow{values: c.data[c.idx]}
}

type mapRow map[string]interface{}

type testDataTable struct{}

func (t testDataTable) NewCursor() (Cursor, error) {
	return &testDataCursor{idx: -1, data: testData}, nil
}

func TestExecutor(t *testing.T) {
	query := "SELECT * WHERE id > 2"
	exec := NewExecutor(testDataTable{})

	q, err := Parse(query)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(exec.Execute(q))
}
