// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testUserdataManifest = `{
  "vmconfig": {
    "version": "2026.01.01",
    "kind": "knc",
    "os": { "name": "debian-13", "arch": "arm64", "image": {} },
    "dependencies": {
      "runc": {
        "version": "1.2.3",
        "url": "https://example.com/runc",
        "type": "binary",
        "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      }
    },
    "images": []
  }
}`

// TestRenderUserdata verifies provider inputs produce canonical userdata.
func TestRenderUserdata(t *testing.T) {
	content, err := renderUserdata(userdataModel{
		ManifestJSON:               types.StringValue(testUserdataManifest),
		DepsMirrorURL:              types.StringValue("https://deps.podplane.dev"),
		ProviderKind:               types.StringValue("aws"),
		AWSAccountID:               types.StringValue("123456789012"),
		ImmutableSSHAuthorizedKeys: types.StringValue("ssh-ed25519 AAAAimmutable-one\nssh-ed25519 AAAAimmutable-two admin's-key"),
		EnableSSM:                  types.BoolValue(true),
	})
	if err != nil {
		t.Fatalf("renderUserdata: %v", err)
	}
	for _, want := range []string{
		"# Provider: aws",
		"AWS_ACCOUNT_ID='123456789012'",
		"IMMUTABLE_SSH_AUTHORIZED_KEYS='ssh-ed25519 AAAAimmutable-one\nssh-ed25519 AAAAimmutable-two admin'\\''s-key'",
		"https://deps.podplane.dev/vmconfig/artifacts/runc/1.2.3/runc",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered userdata missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "\nSSH_AUTHORIZED_KEYS=") {
		t.Fatalf("mutable SSH keys must not be rendered into userdata:\n%s", content)
	}
	if strings.Contains(content, "%{ if var.enable_ssm") {
		t.Fatalf("Terraform template directives must be resolved by the provider:\n%s", content)
	}
}

// TestRenderUserdataRejectsInvalidManifest verifies invalid manifests surface
// a rendering error.
func TestRenderUserdataRejectsInvalidManifest(t *testing.T) {
	_, err := renderUserdata(userdataModel{
		ManifestJSON: types.StringValue("{"),
		ProviderKind: types.StringValue("aws"),
	})
	if err == nil {
		t.Fatal("expected invalid manifest error")
	}
}

// TestUserdataDataSourceSchema verifies the public Terraform data source
// contract.
func TestUserdataDataSourceSchema(t *testing.T) {
	var resp datasource.SchemaResponse
	(&userdataDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	for _, name := range []string{
		"manifest_json",
		"deps_mirror_url",
		"provider_kind",
		"aws_account_id",
		"google_project_id",
		"immutable_ssh_authorized_keys",
		"enable_ssm",
		"content",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Fatalf("schema missing %q", name)
		}
	}
	content, ok := resp.Schema.Attributes["content"].(datasourceschema.StringAttribute)
	if !ok || !content.Computed {
		t.Fatalf("content attribute must be a computed string: %#v", resp.Schema.Attributes["content"])
	}
}
