// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*netsySeedS3Resource)(nil)

type netsySeedS3Resource struct{}

type netsySeedS3Model struct {
	ID                types.String `tfsdk:"id"`
	ClusterConfigPath types.String `tfsdk:"cluster_config_path"`
	SeedPath          types.String `tfsdk:"seed_path"`
	ValuesFile        types.String `tfsdk:"values_file"`
	ValuesContent     types.String `tfsdk:"values_content"`
	Bucket            types.String `tfsdk:"bucket"`
	Prefix            types.String `tfsdk:"prefix"`
	Region            types.String `tfsdk:"region"`
	Profile           types.String `tfsdk:"profile"`
}

// NewNetsySeedS3Resource returns the S3 Netsy seed resource implementation.
func NewNetsySeedS3Resource() resource.Resource {
	return &netsySeedS3Resource{}
}

// Metadata returns the resource type name.
func (r *netsySeedS3Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_netsy_seed_s3"
}

// Schema returns the resource schema.
func (r *netsySeedS3Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{Computed: true},
			"cluster_config_path": resourceschema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"seed_path": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values_file": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Path to a user-authored YAML or JSON values file, applied last.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"values_content": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Inline YAML or JSON values, applied before values_file.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket": resourceschema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"prefix": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile": resourceschema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create generates the Netsy snapshot, uploads it to S3, and records state.
func (r *netsySeedS3Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan netsySeedS3Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ValuesContent.IsUnknown() {
		resp.Diagnostics.AddError("Resolve Netsy seed values", "values_content must be known before creating the Netsy seed")
		return
	}
	if plan.ValuesFile.IsUnknown() {
		resp.Diagnostics.AddError("Resolve Netsy seed values", "values_file must be known before creating the Netsy seed")
		return
	}
	opts := seedOptionsFromS3Model(plan)
	doSeed, err := shouldSeed(opts)
	if err != nil {
		resp.Diagnostics.AddError("Read Netsy seed config", err.Error())
		return
	}
	if doSeed {
		if err := checkSnapshotTargetS3(ctx, opts); err != nil {
			resp.Diagnostics.AddError("Check Netsy seed target", err.Error())
			return
		}
		seedPath, cleanup, err := resolveSeedPath(ctx, opts)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			resp.Diagnostics.AddError("Resolve Netsy seed", err.Error())
			return
		}
		path, snapshotCleanup, err := generateNetsySeedSnapshot(ctx, opts, seedPath)
		if snapshotCleanup != nil {
			defer snapshotCleanup()
		}
		if err != nil {
			resp.Diagnostics.AddError("Generate Netsy snapshot", err.Error())
			return
		}
		if err := uploadSnapshotS3(ctx, opts, path); err != nil {
			resp.Diagnostics.AddError("Upload Netsy snapshot to S3", err.Error())
			return
		}
	}
	plan.ID = types.StringValue("s3://" + opts.Bucket + "/" + opts.objectKey())
	plan.Prefix = types.StringValue(opts.Prefix)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read keeps the stored seed resource state.
func (r *netsySeedS3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state netsySeedS3Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates Terraform state; seed-affecting fields require replacement.
func (r *netsySeedS3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan netsySeedS3Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	opts := seedOptionsFromS3Model(plan)
	plan.ID = types.StringValue("s3://" + opts.Bucket + "/" + opts.objectKey())
	plan.Prefix = types.StringValue(opts.Prefix)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete leaves remote Netsy state intact and removes Terraform state.
func (r *netsySeedS3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

// seedOptionsFromS3Model converts Terraform S3 resource state to seed options.
func seedOptionsFromS3Model(model netsySeedS3Model) SeedOptions {
	return SeedOptions{
		ClusterConfigPath: stringOr(model.ClusterConfigPath, ""),
		SeedPath:          stringOr(model.SeedPath, ""),
		ValuesFile:        stringOr(model.ValuesFile, ""),
		ValuesContent:     stringOr(model.ValuesContent, ""),
		Bucket:            stringOr(model.Bucket, ""),
		Prefix:            stringOr(model.Prefix, ""),
		Region:            stringOr(model.Region, ""),
		Profile:           stringOr(model.Profile, ""),
	}
}
