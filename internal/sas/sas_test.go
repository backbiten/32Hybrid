package sas_test

import (
	"strings"
	"testing"
	"time"

	"github.com/backbiten/32Hybrid/internal/sas"
)

func TestBlobSASURL_Smoke(t *testing.T) {
	// Use a valid-looking base64 key (32 bytes = 256-bit, base64-encoded).
	// In real use the key is 512 bits; this is just enough for HMAC-SHA256.
	fakeKey := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" // 32 bytes base64

	params := sas.Params{
		AccountName: "testaccount",
		AccountKey:  fakeKey,
		Container:   "uploads",
		BlobName:    "uploads/abc123/app.exe",
		Permissions: "r",
		Expiry:      time.Now().Add(time.Hour),
	}

	url, err := sas.BlobSASURL(params)
	if err != nil {
		t.Fatalf("BlobSASURL returned error: %v", err)
	}
	if !strings.HasPrefix(url, "https://testaccount.blob.core.windows.net/") {
		t.Errorf("unexpected URL prefix: %s", url)
	}
	for _, must := range []string{"sv=2020-12-06", "sp=r", "sr=b", "spr=https", "sig="} {
		if !strings.Contains(url, must) {
			t.Errorf("URL missing expected query param %q: %s", must, url)
		}
	}
}

func TestBlobSASURL_MissingFields(t *testing.T) {
	cases := []sas.Params{
		{AccountKey: "key", Container: "c", BlobName: "b", Permissions: "r", Expiry: time.Now().Add(time.Hour)},
		{AccountName: "a", Container: "c", BlobName: "b", Permissions: "r", Expiry: time.Now().Add(time.Hour)},
		{AccountName: "a", AccountKey: "key", BlobName: "b", Permissions: "r", Expiry: time.Now().Add(time.Hour)},
		{AccountName: "a", AccountKey: "key", Container: "c", Permissions: "r", Expiry: time.Now().Add(time.Hour)},
		{AccountName: "a", AccountKey: "key", Container: "c", BlobName: "b", Permissions: "r"}, // zero expiry
	}
	for i, p := range cases {
		_, err := sas.BlobSASURL(p)
		if err == nil {
			t.Errorf("case %d: expected error for incomplete params, got nil", i)
		}
	}
}
