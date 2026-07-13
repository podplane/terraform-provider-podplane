// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSeedOptionsObjectKey verifies bootstrap.netsy is always used under the
// optional normalized prefix.
func TestSeedOptionsObjectKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "empty", prefix: "", want: "bootstrap.netsy"},
		{name: "plain", prefix: "netsy", want: "netsy/bootstrap.netsy"},
		{name: "slashes", prefix: "/netsy/", want: "netsy/bootstrap.netsy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeedOptions{Prefix: tt.prefix}.objectKey()
			if got != tt.want {
				t.Fatalf("objectKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestShouldSeed verifies seed generation is skipped only when the cluster seed
// is omitted or explicitly none.
func TestShouldSeed(t *testing.T) {
	tests := []struct {
		name     string
		seedJSON string
		opts     SeedOptions
		want     bool
	}{
		{name: "explicit seed path", opts: SeedOptions{SeedPath: "/tmp/custom.netsy"}, want: true},
		{name: "omitted seed", seedJSON: ``, want: false},
		{name: "empty seed", seedJSON: `"seed": {}`, want: false},
		{name: "none seed", seedJSON: `"seed": {"name":"none"}`, want: false},
		{name: "recommended seed", seedJSON: `"seed": {"name":"recommended","version":"v1.0.0","digest":"sha512:00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}`, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			if opts.SeedPath == "" {
				opts.ClusterConfigPath = writeClusterConfig(t, tt.seedJSON)
			}
			got, err := shouldSeed(opts)
			if err != nil {
				t.Fatalf("shouldSeed() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("shouldSeed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResolveExplicitSeedPath verifies custom local seeds honor the digest
// pinned in cluster config.
func TestResolveExplicitSeedPath(t *testing.T) {
	contents := []byte("custom seed")
	path := filepath.Join(t.TempDir(), "custom.netsy")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write custom seed: %v", err)
	}
	sum := sha512.Sum512(contents)
	digest := "sha512:" + hex.EncodeToString(sum[:])
	resolved, cleanup, err := resolveExplicitSeedPath(context.Background(), path, digest)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatalf("resolveExplicitSeedPath returned error: %v", err)
	}
	if resolved != path {
		t.Fatalf("resolved path = %q, want %q", resolved, path)
	}
	if _, _, err := resolveExplicitSeedPath(context.Background(), path, "sha512:"+strings.Repeat("0", 128)); err == nil {
		t.Fatal("resolveExplicitSeedPath returned nil for mismatched digest")
	}
}

// writeClusterConfig writes a minimal cluster config with optional seed JSON.
func writeClusterConfig(t *testing.T, seedJSON string) string {
	t.Helper()
	seedBlock := ""
	if seedJSON != "" {
		seedBlock = "," + seedJSON
	}
	path := filepath.Join(t.TempDir(), "podplane.cluster.jsonc")
	contents := `{
  "cluster": {
    "id": "testcluster",
    "oidc": {"issuer_url": "https://oidc.example.com"}` + seedBlock + `
  }
}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write cluster config: %v", err)
	}
	return path
}
