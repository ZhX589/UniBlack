package domain

import "testing"

func TestNormalizePagination(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		page, size, wantPage, wantSize int
	}{
		{"missing", 0, 0, 1, 20},
		{"negative", -3, -10, 1, 20},
		{"oversized resets rather than clamps", 3, 101, 3, 20},
		{"minimum", 1, 1, 1, 1},
		{"maximum", 2, 100, 2, 100},
		{"valid", 4, 35, 4, 35},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, size := NormalizePagination(tc.page, tc.size)
			if page != tc.wantPage || size != tc.wantSize {
				t.Fatalf("NormalizePagination(%d, %d) = (%d, %d), want (%d, %d)", tc.page, tc.size, page, size, tc.wantPage, tc.wantSize)
			}
			againPage, againSize := NormalizePagination(page, size)
			if againPage != page || againSize != size {
				t.Fatal("normalization is not idempotent")
			}
		})
	}
}
