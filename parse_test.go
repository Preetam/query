package query

import "testing"

func TestParser(t *testing.T) {
	validQueries := []string{
		"SELECT *",
		"SELECT * WHERE foo = 1",
		"SELECT * WHERE foo = 1, bar = 2",
		"SELECT * WHERE foo = 1, bar = 2 GROUP BY foo",
		"SELECT * WHERE foo = 1, bar = 2 GROUP BY foo ORDER BY bar",
		"SELECT * WHERE foo = 1, bar = 2 ORDER BY foo",
		"SELECT * WHERE foo = 1, bar = 2 LIMIT 10",
		"SELECT * WHERE foo = 1, bar = 2 ORDER BY foo DESC",
	}

	for _, q := range validQueries {
		_, err := Parse(q)
		if err != nil {
			t.Error(q, err)
		}
	}
}
