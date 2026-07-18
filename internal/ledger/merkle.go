package ledger

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

// MerkleTree is an append-only-friendly Merkle tree over record hashes. It
// provides tamper-evident inclusion proofs so a verifier can prove a single
// record belongs to the ledger without downloading the whole chain.
type MerkleTree struct {
	// levels[0] holds the leaf hashes (one per record, in id order).
	// levels[len-1] holds a single root hash.
	levels [][]string
}

// NewMerkleTree builds a Merkle tree from the given ordered leaf hashes.
// An empty slice yields a tree whose root is the SHA-256 of the empty string.
func NewMerkleTree(leaves []string) *MerkleTree {
	t := &MerkleTree{}
	if len(leaves) == 0 {
		empty := sha256.Sum256([]byte(""))
		t.levels = [][]string{{hex.EncodeToString(empty[:])}}
		return t
	}

	level := make([]string, len(leaves))
	copy(level, leaves)
	t.levels = append(t.levels, level)

	for len(t.levels[len(t.levels)-1]) > 1 {
		upper := t.levels[len(t.levels)-1]
		next := make([]string, 0, (len(upper)+1)/2)
		for i := 0; i < len(upper); i += 2 {
			left := upper[i]
			var right string
			if i+1 < len(upper) {
				right = upper[i+1]
			} else {
				right = left // odd node duplicated
			}
			combined := sha256.Sum256([]byte(left + "|" + right))
			next = append(next, hex.EncodeToString(combined[:]))
		}
		t.levels = append(t.levels, next)
	}

	return t
}

// Root returns the Merkle root hash.
func (t *MerkleTree) Root() string {
	return t.levels[len(t.levels)-1][0]
}

// MerkleProof is an inclusion proof for a single leaf.
type MerkleProof struct {
	Leaf       string   `json:"leaf"`
	Index      int      `json:"index"`
	Root       string   `json:"root"`
	Siblings   []string `json:"siblings"`   // sibling hashes bottom-up
	IsRight    []bool   `json:"is_right"`   // true if sibling is on the right
}

// GenerateProof returns an inclusion proof for the leaf at the given index.
func (t *MerkleTree) GenerateProof(index int) (MerkleProof, error) {
	if index < 0 || index >= len(t.levels[0]) {
		return MerkleProof{}, errors.New("merkle: index out of range")
	}

	proof := MerkleProof{
		Leaf:     t.levels[0][index],
		Index:    index,
		Root:     t.Root(),
		Siblings: []string{},
		IsRight:  []bool{},
	}

	idx := index
	for level := 0; level < len(t.levels)-1; level++ {
		current := t.levels[level]
		var sibling string
		var isRight bool
		if idx%2 == 0 {
			// current is left, sibling is right (or duplicated if last)
			if idx+1 < len(current) {
				sibling = current[idx+1]
			} else {
				sibling = current[idx]
			}
			isRight = true
		} else {
			sibling = current[idx-1]
			isRight = false
		}
		proof.Siblings = append(proof.Siblings, sibling)
		proof.IsRight = append(proof.IsRight, isRight)
		idx /= 2
	}

	return proof, nil
}

// VerifyProof checks that the proof correctly derives the given root.
func VerifyProof(proof MerkleProof) bool {
	hash := proof.Leaf
	idx := proof.Index
	for i := 0; i < len(proof.Siblings); i++ {
		sib := proof.Siblings[i]
		var combined string
		if proof.IsRight[i] {
			combined = hash + "|" + sib
		} else {
			combined = sib + "|" + hash
		}
		sum := sha256.Sum256([]byte(combined))
		hash = hex.EncodeToString(sum[:])
		idx /= 2
	}
	return hash == proof.Root
}

// SignedRoot is a ledger root hash signed by an ed25519 key. It lets external
// parties trust the ledger state at a point in time without re-verifying the
// entire chain.
type SignedRoot struct {
	Root      string `json:"root"`
	Index     int64  `json:"index"`     // last record id covered by the root
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"` // hex-encoded ed25519 signature
	PublicKey string `json:"public_key"`
}

// SignRoot signs a Merkle root with the given private key.
func SignRoot(root string, index, ts int64, priv ed25519.PrivateKey) SignedRoot {
	msg := []byte(root + "|" + itoa(index) + "|" + itoa(ts))
	sig := ed25519.Sign(priv, msg)
	pub := priv.Public().(ed25519.PublicKey)
	return SignedRoot{
		Root:      root,
		Index:     index,
		Timestamp: ts,
		Signature: hex.EncodeToString(sig),
		PublicKey: hex.EncodeToString(pub),
	}
}

// Verify checks the signature on a SignedRoot.
func (s SignedRoot) Verify() bool {
	pub, err := hex.DecodeString(s.PublicKey)
	if err != nil {
		return false
	}
	sig, err := hex.DecodeString(s.Signature)
	if err != nil {
		return false
	}
	msg := []byte(s.Root + "|" + itoa(s.Index) + "|" + itoa(s.Timestamp))
	return ed25519.Verify(pub, msg, sig)
}

// BuildMerkleTreeFromRecords constructs a Merkle tree from ledger records,
// using their stored hash chain values as leaves (in id order).
func BuildMerkleTreeFromRecords(records []Record) *MerkleTree {
	// Ensure deterministic ordering by id.
	sorted := make([]Record, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	leaves := make([]string, len(sorted))
	for i, r := range sorted {
		leaves[i] = r.Hash
	}
	return NewMerkleTree(leaves)
}

// itoa is a tiny int64->string helper to avoid importing strconv everywhere.
func itoa(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
