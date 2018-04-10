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

func BenchmarkParser(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Parse("SELECT a, b, min(c), sum(d) WHERE a < 1, b < 2, c < 3 GROUP BY a, b ORDER BY min(c) DESC LIMIT 10")
	}
}
