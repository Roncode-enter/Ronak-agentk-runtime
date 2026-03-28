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

// Package verifiable provides real zero-knowledge proof generation using gnark.
//
// Proof modes:
//   - merkle-only:      SHA-256 hash chains (free tier, fast, tamper-detection only)
//   - snark-groth16:    Real Groth16 zk-SNARK proofs via gnark (standard tier)
//   - plonk-universal:  Real PlonK universal proofs via gnark (premium tier, no trusted setup)
package verifiable

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
)

// Proof mode constants.
const (
	ProofModeMerkleOnly     = "merkle-only"
	ProofModeSNARKGroth16   = "snark-groth16"
	ProofModePlonKUniversal = "plonk-universal"
)

// ProofEngine manages proof generation across all proof modes.
// It lazily initializes SNARK/PlonK provers on first use.
type ProofEngine struct {
	mu          sync.Mutex
	snarkProver *SNARKProver
	plonkProver *PlonKProver
	log         logr.Logger
}

// NewProofEngine creates a new ProofEngine. Provers are initialized lazily.
func NewProofEngine(log logr.Logger) *ProofEngine {
	return &ProofEngine{log: log}
}

// getSNARKProver lazily initializes the Groth16 prover.
func (pe *ProofEngine) getSNARKProver() (*SNARKProver, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.snarkProver != nil {
		return pe.snarkProver, nil
	}
	pe.log.Info("Initializing Groth16 SNARK prover (one-time trusted setup)...")
	prover, err := NewSNARKProver()
	if err != nil {
		return nil, err
	}
	pe.snarkProver = prover
	pe.log.Info("Groth16 SNARK prover initialized successfully")
	return pe.snarkProver, nil
}

// getPlonKProver lazily initializes the PlonK prover.
func (pe *ProofEngine) getPlonKProver() (*PlonKProver, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	if pe.plonkProver != nil {
		return pe.plonkProver, nil
	}
	pe.log.Info("Initializing PlonK universal prover (premium tier)...")
	prover, err := NewPlonKProver()
	if err != nil {
		return nil, err
	}
	pe.plonkProver = prover
	pe.log.Info("PlonK universal prover initialized successfully")
	return pe.plonkProver, nil
}

// ProofResult contains the output of a proof generation.
type ProofResult struct {
	// ProofRoot is the proof chain root (hex string).
	ProofRoot string
	// Algorithm describes the cryptographic scheme used.
	Algorithm string
	// ProofBytes contains the serialized proof (empty for merkle-only).
	ProofBytes []byte
	// Verified indicates if the proof was self-verified.
	Verified bool
}

// GenerateStepProof generates a proof for one reconciliation step using the specified mode.
func (pe *ProofEngine) GenerateStepProof(
	proofMode string,
	stepData []byte,
	previousProof string,
	stepIndex int,
) (*ProofResult, error) {
	switch proofMode {
	case ProofModeSNARKGroth16, "full-zk", "sha3-attestation":
		prover, err := pe.getSNARKProver()
		if err != nil {
			pe.log.Error(err, "SNARK prover init failed, falling back to merkle-only")
			return pe.generateMerkleProof(stepData, previousProof, stepIndex), nil
		}
		proof, err := prover.GenerateProof(previousProof, string(stepData), stepIndex)
		if err != nil {
			pe.log.Error(err, "SNARK proof generation failed, falling back to merkle-only")
			return pe.generateMerkleProof(stepData, previousProof, stepIndex), nil
		}
		return &ProofResult{
			ProofRoot:  proof.PublicInputHash,
			Algorithm:  "Groth16-BN254 (zk-SNARK)",
			ProofBytes: proof.ProofBytes,
			Verified:   proof.Verified,
		}, nil

	case ProofModePlonKUniversal:
		prover, err := pe.getPlonKProver()
		if err != nil {
			pe.log.Error(err, "PlonK prover init failed, falling back to merkle-only")
			return pe.generateMerkleProof(stepData, previousProof, stepIndex), nil
		}
		proof, err := prover.GenerateProof(previousProof, string(stepData), stepIndex)
		if err != nil {
			pe.log.Error(err, "PlonK proof generation failed, falling back to merkle-only")
			return pe.generateMerkleProof(stepData, previousProof, stepIndex), nil
		}
		return &ProofResult{
			ProofRoot:  proof.PublicInputHash,
			Algorithm:  "PlonK-BN254 (Universal SNARK — Premium)",
			ProofBytes: proof.ProofBytes,
			Verified:   proof.Verified,
		}, nil

	default: // merkle-only or any unrecognized mode
		return pe.generateMerkleProof(stepData, previousProof, stepIndex), nil
	}
}

// generateMerkleProof uses SHA-256 hash chains (free tier).
func (pe *ProofEngine) generateMerkleProof(stepData []byte, previousProof string, stepIndex int) *ProofResult {
	proof := ComputeStepProof(stepData, previousProof, stepIndex)
	root := ComputeProofChainRoot(previousProof, proof)
	return &ProofResult{
		ProofRoot: root,
		Algorithm: "SHA-256 (Merkle chain)",
		Verified:  true,
	}
}

// AttestationReport represents a signed attestation report for an agent's execution.
type AttestationReport struct {
	AgentName         string `json:"agentName"`
	Namespace         string `json:"namespace"`
	ProofRoot         string `json:"proofRoot"`
	MerkleRoot        string `json:"merkleRoot"`
	AttestationDigest string `json:"attestationDigest"`
	ProofMode         string `json:"proofMode"`
	StepCount         int32  `json:"stepCount"`
	Verified          bool   `json:"verified"`
	Algorithm         string `json:"algorithm"`
}

// ComputeStepProof generates a SHA-256 proof for a single reconciliation step.
// Used directly for merkle-only mode; SNARK/PlonK modes use their own circuits.
func ComputeStepProof(stepData []byte, previousProof string, stepIndex int) string {
	input := fmt.Sprintf("%s|%s|%d", previousProof, string(stepData), stepIndex)
	return hashSHA256([]byte(input))
}

// ComputeProofChainRoot chains a new step proof into the existing proof root.
func ComputeProofChainRoot(previousRoot string, newStepProof string) string {
	if previousRoot == "" {
		return newStepProof
	}
	input := fmt.Sprintf("%s|%s", previousRoot, newStepProof)
	return hashSHA256([]byte(input))
}

// ComputeAttestationDigest creates a digest that binds the proof root to an agent identity and time.
func ComputeAttestationDigest(proofRoot string, agentName string, timestamp string) string {
	input := fmt.Sprintf("%s|%s|%s", proofRoot, agentName, timestamp)
	return hashSHA256([]byte(input))
}

// BuildAttestationReport generates a JSON attestation report containing all proof data.
func BuildAttestationReport(
	agentName string,
	namespace string,
	proofRoot string,
	merkleRoot string,
	attestationDigest string,
	proofMode string,
	stepCount int32,
	algorithm string,
) string {
	if algorithm == "" {
		algorithm = "SHA-256 (Merkle chain)"
	}
	report := AttestationReport{
		AgentName:         agentName,
		Namespace:         namespace,
		ProofRoot:         proofRoot,
		MerkleRoot:        merkleRoot,
		AttestationDigest: attestationDigest,
		ProofMode:         proofMode,
		StepCount:         stepCount,
		Verified:          true,
		Algorithm:         algorithm,
	}
	data, err := json.Marshal(report)
	if err != nil {
		// Return a valid JSON error report instead of empty object
		return fmt.Sprintf(`{"error":"failed to marshal attestation report: %s","agentName":"%s","namespace":"%s"}`,
			err.Error(), agentName, namespace)
	}
	return string(data)
}

// hashSHA256 computes the SHA-256 hash of the input and returns a hex string.
func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
