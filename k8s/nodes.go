// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package k8s

import (
	"github.com/steadybit/steadybit-debug/config"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
)

func AddKubernetesNodesInformation(cfg *config.Config) {
	pathForNodes := filepath.Join(cfg.OutputPath, "nodes")

	ForEachNode(cfg, func(node *v1.Node) {
		pathForNode := filepath.Join(pathForNodes, node.Name)
		AddDescription(cfg, filepath.Join(pathForNode, "description.txt"), "node", node.Namespace, node.Name)
		AddConfig(cfg, filepath.Join(pathForNode, "config.yaml"), "node", node.Namespace, node.Name)
	})
}
