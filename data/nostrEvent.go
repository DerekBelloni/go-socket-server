package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

type NostrEvent struct {
	ID        string     `json:"id,omitempty"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      float64    `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig,omitempty"`
}

func (e *NostrEvent) GenerateId() error {
	// maybe move to its own method, serialize()
	serialized, err := json.Marshal([]interface{}{
		0,
		e.PubKey,
		e.CreatedAt,
		e.Kind,
		e.Tags,
		e.Content,
	})

	if err != nil {
		return fmt.Errorf("error serializing event: %v", err)
	}

	hash := sha256.Sum256(serialized)
	e.ID = hex.EncodeToString(hash[:])

	return nil
}

func (e *NostrEvent) SignEvent(privateKeyHex string) error {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key hex: %w", err)
	}

	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

	idBytes, err := hex.DecodeString(e.ID)
	if err != nil {
		return fmt.Errorf("invalid event ID: %w", err)
	}

	sig, err := schnorr.Sign(privateKey, idBytes)
	if err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}

func (e *NostrEvent) VerifySignature() (bool, error) {
	pubKey, err := hex.DecodeString(e.PubKey)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %w", err)
	}

	sig, err := hex.DecodeString(e.Sig)
	if err != nil {
		return false, fmt.Errorf("invalid signature: %w", err)
	}

	idBytes, err := hex.DecodeString(e.ID)
	if err != nil {
		return false, fmt.Errorf("invalid event ID: %w", err)
	}

	parsedPubKey, err := schnorr.ParsePubKey(pubKey)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	parsedSig, err := schnorr.ParseSignature(sig)
	if err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	return parsedSig.Verify(idBytes, parsedPubKey), nil
}
