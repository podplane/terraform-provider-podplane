// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

// TestWorkloadCAKeySchemaContainsMetadataOnly verifies that private key bytes
// cannot enter configuration or Terraform state.
func TestWorkloadCAKeySchemaContainsMetadataOnly(t *testing.T) {
	schema := resourceSchema(t, &workloadCAKeyResource{})
	for _, name := range []string{"provider", "key_prefix"} {
		assertStringAttribute(t, schema, name, true, false)
	}
	for _, name := range []string{"region", "profile", "project"} {
		assertStringAttribute(t, schema, name, false, true)
	}
	for _, forbidden := range []string{"value", "private_key", "private_key_pem", "secret", "secret_value"} {
		if _, ok := schema.Attributes[forbidden]; ok {
			t.Fatalf("schema exposes forbidden key attribute %q", forbidden)
		}
	}
}
