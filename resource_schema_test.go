// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestNetsySeedS3Schema verifies the S3 resource exposes prefix, not key.
func TestNetsySeedS3Schema(t *testing.T) {
	schema := resourceSchema(t, &netsySeedS3Resource{})
	assertStringAttribute(t, schema, "cluster_config_path", true, false)
	assertStringAttribute(t, schema, "seed_path", false, true)
	assertStringAttribute(t, schema, "values_path", false, true)
	assertStringAttribute(t, schema, "bucket", true, false)
	assertStringAttribute(t, schema, "prefix", false, true)
	assertStringAttribute(t, schema, "region", false, true)
	assertStringAttribute(t, schema, "profile", false, true)
	if _, ok := schema.Attributes["key"]; ok {
		t.Fatalf("schema unexpectedly contains legacy key attribute")
	}
}

// TestNetsySeedGCSSchema verifies the GCS resource exposes prefix, not key.
func TestNetsySeedGCSSchema(t *testing.T) {
	schema := resourceSchema(t, &netsySeedGCSResource{})
	assertStringAttribute(t, schema, "cluster_config_path", true, false)
	assertStringAttribute(t, schema, "seed_path", false, true)
	assertStringAttribute(t, schema, "values_path", false, true)
	assertStringAttribute(t, schema, "bucket", true, false)
	assertStringAttribute(t, schema, "prefix", false, true)
	assertStringAttribute(t, schema, "project", false, true)
	if _, ok := schema.Attributes["key"]; ok {
		t.Fatalf("schema unexpectedly contains legacy key attribute")
	}
}

type schemaResource interface {
	Schema(context.Context, resource.SchemaRequest, *resource.SchemaResponse)
}

// resourceSchema returns a resource schema or fails the test if diagnostics are set.
func resourceSchema(t *testing.T, r schemaResource) resourceschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// assertStringAttribute verifies a string attribute's required/optional flags.
func assertStringAttribute(t *testing.T, schema resourceschema.Schema, name string, required, optional bool) {
	t.Helper()
	attr, ok := schema.Attributes[name]
	if !ok {
		t.Fatalf("schema missing %q attribute", name)
	}
	stringAttr, ok := attr.(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("schema attribute %q has type %T, want StringAttribute", name, attr)
	}
	if stringAttr.Required != required || stringAttr.Optional != optional {
		t.Fatalf("schema attribute %q required/optional = %v/%v, want %v/%v", name, stringAttr.Required, stringAttr.Optional, required, optional)
	}
}
