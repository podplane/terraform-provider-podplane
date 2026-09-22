// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

// TestCanonicalWorkloadCAKeyIdentity verifies canonical names for every backend.
func TestCanonicalWorkloadCAKeyIdentity(t *testing.T) {
	tests := []struct{ provider, prefix, project, want string }{
		{"aws_secrets_manager", "cluster-1", "", "aws-secretsmanager:/cluster-1/workload-ca-key"},
		{"aws_ssm", "team/cluster-1", "", "aws-ssm:/team/cluster-1/workload-ca-key"},
		{"gcp_secret_manager", "cluster_1", "project-1", "gcp-secretmanager:projects/project-1/secrets/cluster_1_workload-ca-key"},
	}
	for _, tc := range tests {
		secret, err := workloadCAKeySecret(workloadCAKeyOptions{Provider: tc.provider, KeyPrefix: tc.prefix, Project: tc.project})
		got := secret.Identity()
		if err != nil || got != tc.want {
			t.Fatalf("identity = %q, %v; want %q", got, err, tc.want)
		}
	}
	for _, opts := range []workloadCAKeyOptions{{Provider: "vault", KeyPrefix: "x"}, {Provider: "aws_ssm", KeyPrefix: "/x"}, {Provider: "gcp_secret_manager", KeyPrefix: "a/b", Project: "p"}, {Provider: "gcp_secret_manager", KeyPrefix: "a.b", Project: "p"}} {
		if _, err := workloadCAKeySecret(opts); err == nil {
			t.Fatalf("identity unexpectedly accepted %+v", opts)
		}
	}
}

// TestGeneratedWorkloadCAKey verifies generated keys are valid and unique.
func TestGeneratedWorkloadCAKey(t *testing.T) {
	a, err := generateWorkloadCAKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateWorkloadCAKey()
	if err != nil {
		t.Fatal(err)
	}
	fa, err := workloadCAKeyFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := workloadCAKeyFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if fa == fb || len(fa) != 64 {
		t.Fatalf("fingerprints = %q and %q", fa, fb)
	}
	if _, err := workloadCAKeyFingerprint(append(a, a...)); err == nil {
		t.Fatal("accepted multiple PEM blocks")
	}
}
