// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package platform

import (
	"context"

	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/output"
	v1 "k8s.io/api/core/v1"
)

// envScript requests /actuator/env from inside the platform container. The platform only answers it with the admin
// secret it writes to a file in the container on every start (X-Steadybit-Admin-Secret), found the way the platform's
// own createAdminToken.sh finds it. The container's shell reads the file and hands the header to curl on stdin, so
// the secret never leaves the pod: only the file's path is recorded in the archive, and it is not in curl's arguments
// either. A platform without the file predates the secret and answers without it.
func envScript() string {
	return `f="${STEADYBIT_MANAGEMENT_ADMIN_SECRET_FILE:-${STEADYBIT_PLATFORM_WORKING_DIR:-/opt/steadybit/platform}/data/admin-secret}"
if [ -r "$f" ]; then
  printf 'X-Steadybit-Admin-Secret: %s\n' "$(cat "$f")" | curl -s -H @- http://localhost:9090/actuator/env
else
  curl -s http://localhost:9090/actuator/env
fi
`
}

func addEnv(cfg *config.Config, pod *v1.Pod, outputPath string) {
	output.AddCommandOutput(context.Background(), output.AddCommandOutputOptions{
		Config:           cfg,
		CommandName:      "kubectl",
		CommandArgs:      []string{"exec", pod.Name, "-n", pod.Namespace, "--", "sh", "-c", envScript()},
		OutputPath:       outputPath,
		ExecutionContext: pod.Namespace + "/" + pod.Name,
	})
}
