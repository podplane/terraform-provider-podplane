// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// uploadSnapshotS3 uploads the snapshot to S3 only when the bucket is empty and
// the object key does not exist.
func uploadSnapshotS3(ctx context.Context, opts SeedOptions, path string) error {
	if opts.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	key := opts.objectKey()
	client, err := newS3Client(ctx, opts)
	if err != nil {
		return err
	}
	if err := requireEmptyS3Prefix(ctx, client, opts.Bucket, opts.objectPrefix()); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open Netsy snapshot %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(opts.Bucket),
		Key:         aws.String(key),
		Body:        file,
		IfNoneMatch: aws.String("*"),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "PreconditionFailed" {
			return fmt.Errorf("S3 object s3://%s/%s already exists; refusing to overwrite Netsy state", opts.Bucket, key)
		}
		return fmt.Errorf("upload Netsy snapshot to s3://%s/%s: %w", opts.Bucket, key, err)
	}
	return nil
}

// checkSnapshotTargetS3 checks that the target S3 prefix has no existing objects.
func checkSnapshotTargetS3(ctx context.Context, opts SeedOptions) error {
	if opts.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	client, err := newS3Client(ctx, opts)
	if err != nil {
		return err
	}
	return requireEmptyS3Prefix(ctx, client, opts.Bucket, opts.objectPrefix())
}

// newS3Client returns an S3 client using the standard AWS SDK credential chain.
func newS3Client(ctx context.Context, opts SeedOptions) (*s3.Client, error) {
	loadOpts := []func(*config.LoadOptions) error{}
	if opts.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(opts.Region))
	}
	if opts.Profile != "" {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(opts.Profile))
	}
	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return s3.NewFromConfig(cfg), nil
}

// requireEmptyS3Prefix fails when any object already exists under prefix.
func requireEmptyS3Prefix(ctx context.Context, client *s3.Client, bucket, prefix string) error {
	in := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	}
	if prefix != "" {
		in.Prefix = aws.String(prefix + "/")
	}
	out, err := client.ListObjectsV2(ctx, in)
	if err != nil {
		return fmt.Errorf("check S3 prefix s3://%s/%s is empty: %w", bucket, prefix, err)
	}
	if out.KeyCount != nil && *out.KeyCount > 0 || len(out.Contents) > 0 {
		return fmt.Errorf("S3 prefix s3://%s/%s already contains Netsy state or other objects", bucket, prefix)
	}
	return nil
}
