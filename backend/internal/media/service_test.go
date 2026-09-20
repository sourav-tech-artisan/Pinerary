package media

import "testing"

func TestAllowedContentType(t *testing.T) {
	if !allowedContentType("image/jpeg") || !allowedContentType("image/png") || !allowedContentType("image/webp") {
		t.Fatal("expected supported image types to be allowed")
	}
	if allowedContentType("image/svg+xml") || allowedContentType("application/octet-stream") {
		t.Fatal("unsafe or unknown content type was allowed")
	}
}
