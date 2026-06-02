// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// main serves the Terraform provider.
func main() {
	if err := providerserver.Serve(context.Background(), NewProvider, providerserver.ServeOpts{
		Address: "registry.terraform.io/podplane/podplane",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
