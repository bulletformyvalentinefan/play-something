package spotify

import (
	"context"
	"testing"
)

func TestEscapeQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"pierce the veil", "pierce+the+veil"},
		{"AC/DC", "AC%2FDC"},
		{"a-b_c.d~e09", "a-b_c.d~e09"},
		{"  hola  ", "++hola++"},
	}
	for _, c := range cases {
		if got := escapeQuery(c.in); got != c.want {
			t.Errorf("escapeQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSearchSpclientEmptyQuery(t *testing.T) {
	for _, q := range []string{"", "   "} {
		res, err := searchSpclient(context.Background(), nil, q, 20)
		if err != nil {
			t.Errorf("searchSpclient(%q) err = %v, want nil", q, err)
		}
		if res != nil {
			t.Errorf("searchSpclient(%q) = %v, want nil", q, res)
		}
	}
}
