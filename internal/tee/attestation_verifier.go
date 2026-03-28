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

package tee

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

// AttestationRequest is sent to the attestation service.
type AttestationRequest struct {
	Nonce     string `json:"nonce"`
	AgentName string `json:"agentName"`
	Namespace string `json:"namespace"`
}

// AttestationQuote is the attestation evidence from the TEE.
type AttestationQuote struct {
	Nonce        string `json:"nonce"`
	AgentName    string `json:"agentName"`
	Namespace    string `json:"namespace"`
	Timestamp    string `json:"timestamp"`
	TEEProvider  string `json:"teeProvider"`
	Measurements string `json:"measurements"`
}

// AttestationResponse is returned by the attestation service.
type AttestationResponse struct {
	Quote     AttestationQuote `json:"quote"`
	Signature string           `json:"signature"`
	PublicKey string           `json:"publicKey"`
	Verified  bool             `json:"verified"`
}

// AttestationResult holds the verified attestation data.
type AttestationResult struct {
	Verified     bool
	Quote        AttestationQuote
	Digest       string
	ErrorMessage string
}

// VerifyAttestation performs a real cryptographic attestation verification.
// It sends a nonce challenge to the attestation service, receives a signed quote,
// and verifies the ECDSA P-256 signature. Only returns Verified=true if the
// cryptographic verification passes.
//
// If trustedPublicKeyPEM is non-empty, the signature is verified against this pre-configured
// root-of-trust key instead of the key from the attestation response. This prevents
// self-signed attestation attacks where a compromised service provides its own key.
func VerifyAttestation(endpoint string, agentName string, namespace string, trustedPublicKeyPEM ...string) (*AttestationResult, error) {
	if endpoint == "" {
		return &AttestationResult{
			Verified:     false,
			ErrorMessage: "no attestation endpoint configured",
		}, nil
	}

	// Generate a cryptographic nonce (32 bytes of randomness)
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	// Send attestation request
	req := AttestationRequest{
		Nonce:     nonce,
		AgentName: agentName,
		Namespace: namespace,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint+"/attest", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return &AttestationResult{
			Verified:     false,
			ErrorMessage: fmt.Sprintf("attestation service unreachable: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &AttestationResult{
			Verified:     false,
			ErrorMessage: fmt.Sprintf("attestation service returned %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	var attestResp AttestationResponse
	if err := json.NewDecoder(resp.Body).Decode(&attestResp); err != nil {
		return nil, fmt.Errorf("failed to decode attestation response: %w", err)
	}

	// Verify the nonce matches (prevents replay attacks)
	if attestResp.Quote.Nonce != nonce {
		return &AttestationResult{
			Verified:     false,
			ErrorMessage: "nonce mismatch — possible replay attack",
		}, nil
	}

	// Determine which public key to use for verification.
	// If a trusted root-of-trust key is provided, use it instead of the response key.
	// This prevents self-signed attestation attacks.
	pubKeyPEM := attestResp.PublicKey
	if len(trustedPublicKeyPEM) > 0 && trustedPublicKeyPEM[0] != "" {
		pubKeyPEM = trustedPublicKeyPEM[0]
	} else if attestResp.PublicKey != "" {
		// WARNING: Using public key from attestation response (self-signed).
		// In production, always configure AttestationPublicKeySecretRef to provide
		// a pre-configured root-of-trust key.
		_ = pubKeyPEM // suppress lint; kept for clarity
	}

	pubKey, err := parsePublicKey(pubKeyPEM)
	if err != nil {
		return &AttestationResult{
			Verified:     false,
			ErrorMessage: fmt.Sprintf("invalid public key: %v", err),
		}, nil
	}

	quoteBytes, _ := json.Marshal(attestResp.Quote)
	verified := verifyECDSASignature(pubKey, quoteBytes, attestResp.Signature)

	// Compute attestation digest
	digest := computeQuoteDigest(attestResp.Quote)

	return &AttestationResult{
		Verified:     verified,
		Quote:        attestResp.Quote,
		Digest:       digest,
		ErrorMessage: func() string {
			if !verified {
				return "ECDSA signature verification failed"
			}
			return ""
		}(),
	}, nil
}

// parsePublicKey parses a PEM-encoded ECDSA public key.
func parsePublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	return ecdsaPub, nil
}

// verifyECDSASignature verifies an ECDSA signature over data.
// Expects signature as hex-encoded r||s where each component is exactly 32 bytes (64 bytes total).
func verifyECDSASignature(pubKey *ecdsa.PublicKey, data []byte, sigHex string) bool {
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 64 {
		return false
	}
	hash := sha256.Sum256(data)
	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])
	return ecdsa.Verify(pubKey, hash[:], r, s)
}

// computeQuoteDigest creates a SHA-256 digest of the attestation quote.
func computeQuoteDigest(quote AttestationQuote) string {
	data, _ := json.Marshal(quote)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
