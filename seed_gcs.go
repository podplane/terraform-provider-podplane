// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// uploadSnapshotGCS uploads the snapshot to GCS only when the bucket is empty
// and the object name does not exist.
func uploadSnapshotGCS(ctx context.Context, opts SeedOptions, path string) error {
	if opts.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	key := opts.objectKey()
	client, err := newGCSClient(ctx, opts)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	if err := requireEmptyGCSPrefix(ctx, client, opts.Bucket, opts.objectPrefix()); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open Netsy snapshot %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	obj := client.Bucket(opts.Bucket).Object(key).If(storage.Conditions{DoesNotExist: true})
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, file); err != nil {
		_ = w.Close()
		return fmt.Errorf("upload Netsy snapshot to gs://%s/%s: %w", opts.Bucket, key, err)
	}
	if err := w.Close(); err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 412 {
			return fmt.Errorf("GCS object gs://%s/%s already exists; refusing to overwrite Netsy state", opts.Bucket, key)
		}
		return fmt.Errorf("upload Netsy snapshot to gs://%s/%s: %w", opts.Bucket, key, err)
	}
	return nil
}

// checkSnapshotTargetGCS checks that the target GCS prefix has no existing objects.
func checkSnapshotTargetGCS(ctx context.Context, opts SeedOptions) error {
	if opts.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	client, err := newGCSClient(ctx, opts)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	return requireEmptyGCSPrefix(ctx, client, opts.Bucket, opts.objectPrefix())
}

// newGCSClient returns a GCS client using Application Default Credentials.
func newGCSClient(ctx context.Context, opts SeedOptions) (*storage.Client, error) {
	clientOpts := []option.ClientOption{}
	if opts.Project != "" {
		clientOpts = append(clientOpts, option.WithQuotaProject(opts.Project))
	}
	client, err := storage.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	return client, nil
}

// requireEmptyGCSPrefix fails when any object already exists under prefix.
func requireEmptyGCSPrefix(ctx context.Context, client *storage.Client, bucket, prefix string) error {
	query := &storage.Query{}
	if prefix != "" {
		query.Prefix = prefix + "/"
	}
	it := client.Bucket(bucket).Objects(ctx, query)
	_, err := it.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check GCS prefix gs://%s/%s is empty: %w", bucket, prefix, err)
	}
	return fmt.Errorf("GCS prefix gs://%s/%s already contains Netsy state or other objects", bucket, prefix)
}
