package config

import "testing"

func TestNormalizeServiceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"enrichment", "enrichment"},
		{"enrichment-api", "enrichment_api"},
		{"enrichment api", "enrichment_api"},
		{"  enrichment  api  ", "enrichment_api"},
		{"ENRICHMENT-API", "enrichment_api"},
		{"foo.bar/baz", "foo_bar_baz"},
		{"__already__snake__", "already_snake"},
		{"123-start", "svc_123_start"},
		{"---", "service"},
		{"", "service"},
	}

	for _, tt := range tests {
		if got := NormalizeServiceName(tt.in); got != tt.want {
			t.Fatalf("NormalizeServiceName(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}
