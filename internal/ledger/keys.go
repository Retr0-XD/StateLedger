package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
)

// KeyPair is a persisted ed25519 signing key used to sign ledger roots.
type KeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// GenerateKeyPair creates a new ed25519 key pair.
func GenerateKeyPair() KeyPair {
	pub, priv, _ := ed25519.GenerateKey(nil)
	return KeyPair{
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
	}
}

// LoadOrCreateKey loads a key from path, or generates and persists one if the
// file does not exist.
func LoadOrCreateKey(path string) (KeyPair, error) {
	if data, err := os.ReadFile(path); err == nil {
		var kp KeyPair
		if err := json.Unmarshal(data, &kp); err == nil && kp.PrivateKey != "" {
			return kp, nil
		}
	}
	kp := GenerateKeyPair()
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return kp, err
	}
	data, err := json.MarshalIndent(kp, "", "  ")
	if err != nil {
		return kp, err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return kp, err
	}
	return kp, nil
}

// privateKey decodes the hex private key into an ed25519.PrivateKey.
func (k KeyPair) privateKey() (ed25519.PrivateKey, error) {
	raw, err := hex.DecodeString(k.PrivateKey)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key length")
	}
	return ed25519.PrivateKey(raw), nil
}

// SignedRootAt computes the current Merkle root, signs it with the key at the
// given path, and returns the signed root. The index is the last record id.
func (l *Ledger) SignedRootAt(keyPath string, ts int64) (SignedRoot, error) {
	root, err := l.MerkleRoot()
	if err != nil {
		return SignedRoot{}, err
	}
	last, err := l.LastID()
	if err != nil {
		return SignedRoot{}, err
	}
	kp, err := LoadOrCreateKey(keyPath)
	if err != nil {
		return SignedRoot{}, err
	}
	priv, err := kp.privateKey()
	if err != nil {
		return SignedRoot{}, err
	}
	return SignRoot(root, last, ts, priv), nil
}

// LastID returns the maximum record id currently in the ledger.
func (l *Ledger) LastID() (int64, error) {
	var id int64
	row := l.db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM ledger_records")
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// dirOf returns the directory portion of a file path.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
