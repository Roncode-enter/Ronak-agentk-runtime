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
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// ProofChainCircuit defines the R1CS arithmetic circuit for verifiable execution proofs.
// It proves: "I know a secretInput such that MiMC(previousProof || secretInput || stepIndex) == expectedOutput"
// without revealing secretInput. This is a real zero-knowledge proof — the verifier learns
// nothing about the secret input, only that the prover executed the computation correctly.
type ProofChainCircuit struct {
	// Public inputs (visible to verifier)
	PreviousProof  frontend.Variable `gnark:",public"`
	StepIndex      frontend.Variable `gnark:",public"`
	ExpectedOutput frontend.Variable `gnark:",public"`

	// Private input (hidden from verifier — this is the zero-knowledge part)
	SecretInput frontend.Variable
}

// Define specifies the arithmetic constraints of the circuit.
// gnark compiles this into an R1CS (Rank-1 Constraint System) that can be
// proven with Groth16 (SNARK) or PlonK (universal SNARK / premium tier).
func (c *ProofChainCircuit) Define(api frontend.API) error {
	// Create MiMC hash instance (the zk-friendly hash function)
	h, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	// Hash: MiMC(previousProof || secretInput || stepIndex)
	h.Write(c.PreviousProof)
	h.Write(c.SecretInput)
	h.Write(c.StepIndex)
	result := h.Sum()

	// Constrain: the hash output MUST equal the expected output
	// If the prover lies about any input, this constraint fails
	api.AssertIsEqual(result, c.ExpectedOutput)

	return nil
}
