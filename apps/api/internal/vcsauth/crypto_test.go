package vcsauth

import (
	"encoding/base64"
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewCipher(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("ghp_test_token_value")
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("got %q want %q", out, plain)
	}
}

func TestCipherRejectsShortKey(t *testing.T) {
	_, err := NewCipher(base64.StdEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}
