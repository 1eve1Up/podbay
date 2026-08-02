package receipt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// NewDeployID returns a new UUID v4 string for receipt.deploy_id.
// Each call yields a distinct id (not reused across receipt writes).
func NewDeployID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back to zero UUID shape is unacceptable —
		// panic so callers never silently reuse an empty/constant id.
		panic(fmt.Sprintf("receipt: deploy_id rand: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ContractDigestFile returns sha256: hex digest of the file bytes at path.
// Digests the on-disk contract file (not an in-memory expanded contract).
func ContractDigestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("receipt: contract_digest: read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
