package extract

import (
	"context"
	"testing"
)

// [max-file-size-oversized] a file larger than the configured max size is not
// indexed (Indexed=false) and is counted as oversized. The signature carries
// WithMaxFileSize.
func TestMaxFileSizeOversized(t *testing.T) {
	ctx := context.Background()
	// a file just over the 500 KiB max.
	big := make([]byte, 500*1024+1)
	for i := range big {
		big[i] = 'x'
	}
	g, _, err := ExtractFile(ctx, "big.go", string(big), "go", WithMaxFileSize(500*1024))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !g.Oversized {
		t.Fatalf("an oversized file must be marked oversized (counted in files_oversized)")
	}
	// a file under the max is not oversized.
	small := "func ok() int {\n\treturn 1\n}\n"
	g2, _, err := ExtractFile(ctx, "small.go", small, "go", WithMaxFileSize(500*1024))
	if err != nil {
		t.Fatalf("extract small: %v", err)
	}
	if g2.Oversized {
		t.Fatalf("a file under the max must not be marked oversized")
	}
}
