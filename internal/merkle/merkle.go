/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package merkle provides Merkle-tree checkpoint logic for verifiable agent execution.
// Each reconciliation creates a checkpoint (SHA-256 hash of the agent spec + timestamp + count).
// Checkpoints are chained together via a rolling Merkle root — if any checkpoint in the
// chain is tampered with, the root will no longer match, providing an immutable audit trail.
package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ComputeCheckpoint creates a SHA-256 hash from the agent spec JSON, a timestamp string,
// and the checkpoint count. This produces a unique fingerprint for each reconciliation event.
func ComputeCheckpoint(specJSON []byte, timestamp string, count int32) string {
	data := fmt.Sprintf("%s|%s|%d", string(specJSON), timestamp, count)
	return hashBytes([]byte(data))
}

// ComputeMerkleRoot combines the previous Merkle root with a new checkpoint to produce
// a new root. If previousRoot is empty (first reconciliation), the root equals the
// checkpoint itself. Otherwise the new root is SHA256(previousRoot + newCheckpoint).
func ComputeMerkleRoot(previousRoot string, newCheckpoint string) string {
	if previousRoot == "" {
		return newCheckpoint
	}
	combined := fmt.Sprintf("%s|%s", previousRoot, newCheckpoint)
	return hashBytes([]byte(combined))
}

// hashBytes computes the SHA-256 hash of data and returns it as a lowercase hex string.
func hashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
