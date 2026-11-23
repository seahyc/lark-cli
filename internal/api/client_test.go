package api

import (
	"strings"
	"testing"
)

func TestDecodeResponseReportsPlainTextGatewayError(t *testing.T) {
	// The Open API gateway answers unknown routes with plain text; a message
	// URL pasted as an ID used to surface as "invalid character 'p' after
	// top-level value".
	var resp BaseResponse
	err := decodeResponse(404, []byte("404 page not found"), &resp)
	if err == nil {
		t.Fatal("expected an error for a non-JSON body")
	}
	if got := err.Error(); got != "unexpected non-JSON response (HTTP 404): 404 page not found" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestDecodeResponseTruncatesLongBodiesAndNamesEmptyOnes(t *testing.T) {
	var resp BaseResponse
	err := decodeResponse(502, []byte("<html>"+strings.Repeat("x", 500)+"</html>"), &resp)
	if err == nil || !strings.HasSuffix(err.Error(), "...") || len(err.Error()) > 260 {
		t.Fatalf("expected a truncated snippet, got %v", err)
	}
	err = decodeResponse(500, nil, &resp)
	if err == nil || !strings.Contains(err.Error(), "(empty body)") {
		t.Fatalf("expected empty-body marker, got %v", err)
	}
}

func TestDecodeResponseParsesJSON(t *testing.T) {
	var resp BaseResponse
	if err := decodeResponse(200, []byte(`{"code":0,"msg":"ok"}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Fatalf("unexpected decode: %+v", resp)
	}
	if err := decodeResponse(200, []byte("not json"), nil); err != nil {
		t.Fatalf("nil result should skip decoding: %v", err)
	}
}
