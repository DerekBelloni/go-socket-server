package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/schnorr"
)

type NostrEvent struct {
	ID        string     `json:"id,omitempty"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig,omitempty"`
}

func (e *NostrEvent) GenerateId() error {
	eventForHashing := NostrEvent{
		PubKey:    e.PubKey,
		CreatedAt: e.CreatedAt,
		Kind:      e.Kind,
		Tags:      e.Tags,
		Content:   e.Content,
	}

	serialized, err := json.Marshal([]interface{}{
		0,
		eventForHashing.PubKey,
		eventForHashing.CreatedAt,
		eventForHashing.Kind,
		eventForHashing.Tags,
		eventForHashing.Content,
	})

	if err != nil {
		return fmt.Errorf("error serializing event: %v", err)
	}

	hash := sha256.Sum256(serialized)
	e.ID = hex.EncodeToString(hash[:])

	return nil
}

func (e *NostrEvent) SignEvent(privateKeyHex string) error {
	// conver the hex string to bytes
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key hex: %w", err)
	}

	// Make a secp256k1 private key from bytes
	privateKey := secp256k1.PrivKeyFromBytes(privateKeyBytes)

	idBytes, err := hex.DecodeString(e.ID)
	if err != nil {
		return fmt.Errorf("invalide event ID: %w", err)
	}

	sig, err := schnorr.Sign(privateKey, idBytes)
	if err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}
