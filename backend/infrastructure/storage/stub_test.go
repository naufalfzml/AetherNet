package storage

import (
	"bytes"
	"context"
	"testing"
)

func TestStubClientUploadFetch(t *testing.T) {
	client := NewStubClient()
	ctx := context.Background()

	pointer, err := client.UploadBytes(ctx, "text/plain", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.Fetch(ctx, pointer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("unexpected bytes: %q", got)
	}

	if _, err := client.Fetch(ctx, "stub://missing"); err == nil {
		t.Fatal("expected missing pointer error")
	}
}
