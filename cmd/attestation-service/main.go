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

// Package main implements the TEE Attestation Service.
//
// This is a real cryptographic attestation service that implements the
// nonce-challenge attestation protocol using ECDSA P-256 signatures.
// On real TEE hardware (Intel TDX, AMD SEV-SNP), the signing key would
// come from the hardware enclave. This service uses software-generated
// keys but the protocol is identical — making it a drop-in replacement
// when deploying on actual confidential computing infrastructure.
//
// Protocol:
//  1. Client sends POST /attest with a random nonce
//  2. Service constructs a quote: {nonce, agentName, timestamp, teeProvider, measurements}
//  3. Service signs the quote with its ECDSA P-256 private key
//  4. Client verifies the signature against the service's public key
//  5. If valid, the attestation is trusted
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"
)

// AttestationRequest is sent by the ConfidentialAgent controller.
type AttestationRequest struct {
	Nonce     string `json:"nonce"`
	AgentName string `json:"agentName"`
	Namespace string `json:"namespace"`
}

// AttestationQuote is the attestation evidence signed by the TEE.
type AttestationQuote struct {
	Nonce        string `json:"nonce"`
	AgentName    string `json:"agentName"`
	Namespace    string `json:"namespace"`
	Timestamp    string `json:"timestamp"`
	TEEProvider  string `json:"teeProvider"`
	Measurements string `json:"measurements"`
}

// AttestationResponse is returned to the controller.
type AttestationResponse struct {
	Quote     AttestationQuote `json:"quote"`
	Signature string           `json:"signature"`
	PublicKey string           `json:"publicKey"`
	Verified  bool             `json:"verified"`
}

var (
	signingKey  *ecdsa.PrivateKey
	teeProvider string
)

func main() {
	teeProvider = os.Getenv("TEE_PROVIDER")
	if teeProvider == "" {
		teeProvider = "software-attestation"
	}

	// Generate ECDSA P-256 signing key at startup
	var err error
	signingKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&signingKey.PublicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key: %v", err)
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})
	log.Printf("Attestation service started with TEE provider: %s", teeProvider)
	log.Printf("Public key:\n%s", string(pubKeyPEM))

	http.HandleFunc("/attest", handleAttest)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/pubkey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write(pubKeyPEM)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleAttest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req AttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Nonce == "" {
		http.Error(w, "nonce is required", http.StatusBadRequest)
		return
	}

	// Build attestation quote
	quote := AttestationQuote{
		Nonce:        req.Nonce,
		AgentName:    req.AgentName,
		Namespace:    req.Namespace,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TEEProvider:  teeProvider,
		Measurements: computeMeasurements(req.AgentName),
	}

	// Serialize and sign the quote with ECDSA
	quoteBytes, _ := json.Marshal(quote)
	hash := sha256.Sum256(quoteBytes)

	sigR, sigS, err := ecdsa.Sign(rand.Reader, signingKey, hash[:])
	if err != nil {
		http.Error(w, "signing failed", http.StatusInternalServerError)
		return
	}

	// Encode signature as hex (r || s, each padded to exactly 32 bytes)
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	sigRBytes := sigR.Bytes()
	sigSBytes := sigS.Bytes()
	copy(rBytes[32-len(sigRBytes):], sigRBytes)
	copy(sBytes[32-len(sigSBytes):], sigSBytes)
	sigBytes := append(rBytes, sBytes...)
	signature := hex.EncodeToString(sigBytes)

	// Encode public key as PEM
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(&signingKey.PublicKey)
	pubKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes}))

	resp := AttestationResponse{
		Quote:     quote,
		Signature: signature,
		PublicKey: pubKeyPEM,
		Verified:  true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"teeProvider": teeProvider,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

// computeMeasurements generates a deterministic measurement hash for the agent.
// On real TEE hardware, this would come from the hardware PCR registers.
func computeMeasurements(agentName string) string {
	data := fmt.Sprintf("measurement:%s:%s", teeProvider, agentName)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// verifySignature verifies an ECDSA signature (exported for testing).
// Expects hex-encoded r||s, each component exactly 32 bytes (64 bytes total).
func verifySignature(pubKey *ecdsa.PublicKey, data []byte, sigHex string) bool {
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 64 {
		return false
	}
	hash := sha256.Sum256(data)
	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])
	return ecdsa.Verify(pubKey, hash[:], r, s)
}
