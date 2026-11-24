// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCurl records its arguments and what it reads on stdin, in place of the container's curl.
const fakeCurl = `#!/bin/sh
echo "args: $*"
echo "stdin: $(cat)"
`

func runEnvScript(t *testing.T, env ...string) string {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(fakeCurl), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", envScript())
	cmd.Env = append([]string{"PATH=" + bin + ":/usr/bin:/bin"}, env...)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %s\n%s", err, out)
	}
	return string(out)
}

// TestEnvScriptKeepsTheSecretOut is the point of going through a shell in the container: the secret is read there,
// so the recorded command names only the file, and it reaches curl on stdin instead of as an argument.
func TestEnvScriptKeepsTheSecretOut(t *testing.T) {
	workingDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workingDir, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDir, "data", "admin-secret"), []byte("s3cret"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := runEnvScript(t, "STEADYBIT_PLATFORM_WORKING_DIR="+workingDir)

	if !strings.Contains(got, "stdin: X-Steadybit-Admin-Secret: s3cret") {
		t.Errorf("expected the secret header on curl's stdin, got:\n%s", got)
	}
	if strings.Contains(strings.SplitN(got, "\n", 2)[0], "s3cret") {
		t.Errorf("the secret must not be in curl's arguments, got:\n%s", got)
	}
	if strings.Contains(envScript(), "s3cret") {
		t.Error("the recorded command must not contain the secret")
	}
}

func TestEnvScriptWithoutSecretFile(t *testing.T) {
	got := runEnvScript(t, "STEADYBIT_MANAGEMENT_ADMIN_SECRET_FILE="+filepath.Join(t.TempDir(), "missing"))

	if !strings.Contains(got, "args: -s http://localhost:9090/actuator/env") || strings.Contains(got, "-H @-") {
		t.Errorf("expected a plain request for a platform without the secret, got:\n%s", got)
	}
}
