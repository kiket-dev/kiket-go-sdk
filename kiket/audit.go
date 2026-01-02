package kiket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// AuditClient handles blockchain audit verification operations.
type AuditClient struct {
	client *Client
}

// NewAuditClient creates a new audit client.
func NewAuditClient(client *Client) *AuditClient {
	return &AuditClient{client: client}
}

// BlockchainAnchor represents a blockchain anchor containing a batch of audit records.
type BlockchainAnchor struct {
	ID             int64          `json:"id"`
	MerkleRoot     string         `json:"merkle_root"`
	LeafCount      int            `json:"leaf_count"`
	FirstRecordAt  *string        `json:"first_record_at"`
	LastRecordAt   *string        `json:"last_record_at"`
	Network        string         `json:"network"`
	Status         string         `json:"status"`
	TxHash         *string        `json:"tx_hash"`
	BlockNumber    *int64         `json:"block_number"`
	BlockTimestamp *string        `json:"block_timestamp"`
	ConfirmedAt    *string        `json:"confirmed_at"`
	ExplorerURL    *string        `json:"explorer_url"`
	CreatedAt      *string        `json:"created_at"`
	Records        []AnchorRecord `json:"records,omitempty"`
}

// AnchorRecord represents a record within an anchor.
type AnchorRecord struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	LeafIndex   int    `json:"leaf_index"`
	ContentHash string `json:"content_hash"`
}

// BlockchainProof represents a Merkle proof for an audit record.
type BlockchainProof struct {
	RecordID        int64    `json:"record_id"`
	RecordType      string   `json:"record_type"`
	ContentHash     string   `json:"content_hash"`
	AnchorID        int64    `json:"anchor_id"`
	MerkleRoot      string   `json:"merkle_root"`
	LeafIndex       int      `json:"leaf_index"`
	LeafCount       int      `json:"leaf_count"`
	Proof           []string `json:"proof"`
	Network         string   `json:"network"`
	TxHash          *string  `json:"tx_hash"`
	BlockNumber     *int64   `json:"block_number"`
	BlockTimestamp  *string  `json:"block_timestamp"`
	Verified        bool     `json:"verified"`
	VerificationURL *string  `json:"verification_url"`
}

// VerificationResult is the result of a blockchain verification.
type VerificationResult struct {
	Verified           bool    `json:"verified"`
	ProofValid         bool    `json:"proof_valid"`
	BlockchainVerified bool    `json:"blockchain_verified"`
	ContentHash        string  `json:"content_hash"`
	MerkleRoot         string  `json:"merkle_root"`
	LeafIndex          int     `json:"leaf_index"`
	BlockNumber        *int64  `json:"block_number"`
	BlockTimestamp     *string `json:"block_timestamp"`
	Network            *string `json:"network"`
	ExplorerURL        *string `json:"explorer_url"`
	Error              *string `json:"error"`
}

// ListAnchorsOptions are options for listing blockchain anchors.
type ListAnchorsOptions struct {
	Status  string
	Network string
	From    *time.Time
	To      *time.Time
	Page    int
	PerPage int
}

// ListAnchorsResult is the result of listing blockchain anchors.
type ListAnchorsResult struct {
	Anchors    []BlockchainAnchor `json:"anchors"`
	Pagination PaginationInfo     `json:"pagination"`
}

// PaginationInfo contains pagination details.
type PaginationInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ListAnchors lists blockchain anchors for the organization.
func (c *AuditClient) ListAnchors(opts ListAnchorsOptions) (*ListAnchorsResult, error) {
	params := url.Values{}
	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	} else {
		params.Set("page", "1")
	}
	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	} else {
		params.Set("per_page", "25")
	}
	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.Network != "" {
		params.Set("network", opts.Network)
	}
	if opts.From != nil {
		params.Set("from", opts.From.Format(time.RFC3339))
	}
	if opts.To != nil {
		params.Set("to", opts.To.Format(time.RFC3339))
	}

	resp, err := c.client.Get("/api/v1/audit/anchors?" + params.Encode())
	if err != nil {
		return nil, err
	}

	var result ListAnchorsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetAnchor gets details of a specific anchor by merkle root.
func (c *AuditClient) GetAnchor(merkleRoot string, includeRecords bool) (*BlockchainAnchor, error) {
	path := "/api/v1/audit/anchors/" + merkleRoot
	if includeRecords {
		path += "?include_records=true"
	}

	resp, err := c.client.Get(path)
	if err != nil {
		return nil, err
	}

	var anchor BlockchainAnchor
	if err := json.Unmarshal(resp, &anchor); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &anchor, nil
}

// GetProof gets the blockchain proof for a specific audit record.
func (c *AuditClient) GetProof(recordID int64) (*BlockchainProof, error) {
	resp, err := c.client.Get(fmt.Sprintf("/api/v1/audit/records/%d/proof", recordID))
	if err != nil {
		return nil, err
	}

	var proof BlockchainProof
	if err := json.Unmarshal(resp, &proof); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &proof, nil
}

// Verify verifies a blockchain proof via the API.
func (c *AuditClient) Verify(proof *BlockchainProof) (*VerificationResult, error) {
	payload := map[string]interface{}{
		"content_hash": proof.ContentHash,
		"merkle_root":  proof.MerkleRoot,
		"proof":        proof.Proof,
		"leaf_index":   proof.LeafIndex,
		"tx_hash":      proof.TxHash,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.client.Post("/api/v1/audit/verify", body)
	if err != nil {
		return nil, err
	}

	var result VerificationResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ComputeContentHash computes the content hash for a record (for local verification).
func ComputeContentHash(data map[string]interface{}) string {
	// Sort keys for canonical JSON
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make(map[string]interface{})
	for _, k := range keys {
		sorted[k] = data[k]
	}

	canonical, _ := json.Marshal(sorted)
	hash := sha256.Sum256(canonical)
	return "0x" + hex.EncodeToString(hash[:])
}

// VerifyProofLocally verifies a Merkle proof locally without making an API call.
func VerifyProofLocally(contentHash string, proofPath []string, leafIndex int, merkleRoot string) bool {
	current := normalizeHash(contentHash)
	idx := leafIndex

	for _, siblingHex := range proofPath {
		sibling := normalizeHash(siblingHex)
		if idx%2 == 0 {
			current = hashPair(current, sibling)
		} else {
			current = hashPair(sibling, current)
		}
		idx /= 2
	}

	expected := normalizeHash(merkleRoot)
	return bytes.Equal(current, expected)
}

func normalizeHash(h string) []byte {
	if len(h) >= 2 && h[:2] == "0x" {
		h = h[2:]
	}
	decoded, _ := hex.DecodeString(h)
	return decoded
}

func hashPair(left, right []byte) []byte {
	// Sort for consistent ordering
	if bytes.Compare(left, right) > 0 {
		left, right = right, left
	}

	combined := append(left, right...)
	hash := sha256.Sum256(combined)
	return hash[:]
}
