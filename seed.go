// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/podplane/podplane/pkg/netsyseed"
	"github.com/podplane/podplane/pkg/seeds"
)

const netsyBootstrapObject = "bootstrap.netsy"

type SeedOptions struct {
	ClusterConfigPath string
	SeedPath          string
	ValuesPath        string
	Bucket            string
	Prefix            string
	Region            string
	Profile           string
	Project           string
}

// objectKey returns the Netsy bootstrap object key under the configured prefix.
func (opts SeedOptions) objectKey() string {
	return path.Join(opts.objectPrefix(), netsyBootstrapObject)
}

// objectPrefix returns the normalized object prefix used for Netsy state checks.
func (opts SeedOptions) objectPrefix() string {
	return strings.Trim(opts.Prefix, "/")
}

// shouldSeed reports whether this resource should generate and upload a seed.
func shouldSeed(opts SeedOptions) (bool, error) {
	if opts.SeedPath != "" {
		return true, nil
	}
	seed, err := seeds.ReadClusterSeed(opts.ClusterConfigPath)
	if err != nil {
		return false, err
	}
	return seed.Name != seeds.None, nil
}

// resolveSeedPath returns the explicit or downloaded Podplane seed file path.
func resolveSeedPath(ctx context.Context, opts SeedOptions) (seedPath string, cleanup func(), err error) {
	if opts.SeedPath != "" {
		return opts.SeedPath, nil, nil
	}
	depsCacheDir, err := os.MkdirTemp("", "podplane-deps-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary deps cache: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(depsCacheDir) }
	seedPath, err = seeds.ResolveClusterSeedPath(seeds.ResolveClusterOptions{
		Context:           ctx,
		ClusterConfigPath: opts.ClusterConfigPath,
		CacheDir:          depsCacheDir,
	})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return seedPath, cleanup, nil
}

// generateNetsySeedSnapshot generates a temporary Netsy snapshot file from an
// already-resolved seed file path.
func generateNetsySeedSnapshot(ctx context.Context, opts SeedOptions, seedPath string) (path string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "podplane-netsy-*.netsy")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary snapshot: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup = func() {
		_ = os.Remove(tmpPath)
	}
	if err := netsyseed.WriteSnapshot(tmp, netsyseed.SnapshotOptions{
		Context:           ctx,
		ClusterConfigPath: opts.ClusterConfigPath,
		SeedPath:          seedPath,
		ValuesFile:        opts.ValuesPath,
	}); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmpPath, cleanup, nil
}
