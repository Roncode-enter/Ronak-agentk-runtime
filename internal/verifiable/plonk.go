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

package verifiable

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/scs"
	"github.com/consensys/gnark/test/unsafekzg"
)

// PlonKProver handles PlonK universal zero-knowledge proof generation and verification.
// PlonK is a PREMIUM tier proof system — it uses a universal Structured Reference String (SRS)
// that works for ANY circuit, eliminating the per-circuit trusted setup ceremony required by Groth16.
// This makes PlonK more flexible and trustworthy for production deployments.
// Proofs are slightly larger than Groth16 (~400 bytes vs ~128 bytes) but verification
// is still fast (~2ms) and the universal setup is a significant security advantage.
type PlonKProver struct {
	mu  sync.RWMutex
	ccs constraint.ConstraintSystem
	pk  plonk.ProvingKey
	vk  plonk.VerifyingKey
}

// PlonKProof contains the generated PlonK proof and public signals.
type PlonKProof struct {
	// ProofBytes is the serialized PlonK proof.
	ProofBytes []byte
	// PublicInputHash is the MiMC hash of the public inputs.
	PublicInputHash string
	// Verified indicates whether the proof was self-verified after generation.
	Verified bool
	// ProofSystem identifies this as PlonK (premium tier).
	ProofSystem string
}

// NewPlonKProver creates a new PlonK prover with universal setup.
// Unlike Groth16, the SRS can be reused across different circuits.
func NewPlonKProver() (*PlonKProver, error) {
	// Compile the ProofChainCircuit into a Sparse Constraint System (SCS) for PlonK
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, &ProofChainCircuit{})
	if err != nil {
		return nil, fmt.Errorf("failed to compile PlonK circuit: %w", err)
	}

	// Generate the universal SRS (Structured Reference String)
	// In production, this would use a publicly verifiable ceremony
	srs, srsLagrange, err := unsafekzg.NewSRS(ccs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SRS: %w", err)
	}

	// Run PlonK setup with the universal SRS
	pk, vk, err := plonk.Setup(ccs, srs, srsLagrange)
	if err != nil {
		return nil, fmt.Errorf("failed to run PlonK setup: %w", err)
	}

	return &PlonKProver{
		ccs: ccs,
		pk:  pk,
		vk:  vk,
	}, nil
}

// GenerateProof creates a real PlonK zero-knowledge proof (premium tier).
func (p *PlonKProver) GenerateProof(previousProof, secretInput string, stepIndex int) (*PlonKProof, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Compute the expected output
	expectedOutput := computeMiMCHash(previousProof, secretInput, stepIndex)

	prevBig := stringToBigInt(previousProof)
	secretBig := stringToBigInt(secretInput)
	expectedBig := new(big.Int).SetBytes(expectedOutput)

	// Create the witness
	assignment := &ProofChainCircuit{
		PreviousProof:  prevBig,
		StepIndex:      stepIndex,
		ExpectedOutput: expectedBig,
		SecretInput:    secretBig,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	// Generate the PlonK proof
	proof, err := plonk.Prove(p.ccs, p.pk, witness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PlonK proof: %w", err)
	}

	// Serialize the proof
	var proofBuf bytes.Buffer
	_, err = proof.WriteTo(&proofBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	// Self-verify
	publicWitness, err := witness.Public()
	if err != nil {
		return nil, fmt.Errorf("failed to extract public witness: %w", err)
	}

	verified := plonk.Verify(proof, p.vk, publicWitness) == nil

	return &PlonKProof{
		ProofBytes:      proofBuf.Bytes(),
		PublicInputHash: hex.EncodeToString(expectedOutput),
		Verified:        verified,
		ProofSystem:     "PlonK-BN254 (Universal SNARK — Premium Tier)",
	}, nil
}

// VerifyProof verifies a previously generated PlonK proof.
func (p *PlonKProver) VerifyProof(proofBytes []byte, previousProof string, stepIndex int, expectedOutputHex string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proof := plonk.NewProof(ecc.BN254)
	_, err := proof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	expectedBytes, err := hex.DecodeString(expectedOutputHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode expected output: %w", err)
	}

	assignment := &ProofChainCircuit{
		PreviousProof:  stringToBigInt(previousProof),
		StepIndex:      stepIndex,
		ExpectedOutput: new(big.Int).SetBytes(expectedBytes),
	}

	publicWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	err = plonk.Verify(proof, p.vk, publicWitness)
	return err == nil, nil
}

// GetVerifyingKeyBytes serializes the PlonK verifying key.
func (p *PlonKProver) GetVerifyingKeyBytes() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var buf bytes.Buffer
	_, err := p.vk.WriteTo(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
