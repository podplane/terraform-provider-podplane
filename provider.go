// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var providerVersion = "dev"

var _ provider.Provider = (*podplaneProvider)(nil)

type podplaneProvider struct{}

// NewProvider returns a Podplane Terraform provider instance.
func NewProvider() provider.Provider {
	return &podplaneProvider{}
}

// Metadata returns provider type name and version.
func (p *podplaneProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "podplane"
	resp.Version = providerVersion
}

// Schema returns the provider configuration schema.
func (p *podplaneProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

// Configure validates provider configuration.
func (p *podplaneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
}

// DataSources returns provider data sources.
func (p *podplaneProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

// Resources returns provider resources.
func (p *podplaneProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetsySeedS3Resource,
		NewNetsySeedGCSResource,
	}
}
