package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedTrustsOnlyAMatchingChecksum(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "tailwindcss-test")
	body := []byte("not really tailwind")
	sum := sha256.Sum256(body)
	if err := os.WriteFile(bin, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if pinned(bin) {
		t.Error("no sha256sums.txt: pinned")
	}
	sums := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "sha256sums.txt"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sums("0000000000000000000000000000000000000000000000000000000000000000  ./tailwindcss-test\n")
	if pinned(bin) {
		t.Error("a wrong checksum: pinned")
	}
	sums(hex.EncodeToString(sum[:]) + "  ./tailwindcss-other\n")
	if pinned(bin) {
		t.Error("the checksum listed for another file: pinned")
	}
	sums("aaaa  ./x\r\n" + hex.EncodeToString(sum[:]) + "  ./tailwindcss-test\r\n")
	if !pinned(bin) {
		t.Error("its own checksum, CRLF: not pinned")
	}
	if err := os.WriteFile(bin, append(body, '!'), 0o600); err != nil {
		t.Fatal(err)
	}
	if pinned(bin) {
		t.Error("the binary changed after the checksum: pinned")
	}
}
