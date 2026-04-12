package collab

import (
	"encoding/json"
	"strings"
)

// HLC is a Hybrid Logical Clock (Lamport + physical time). Used as the timestamp
// component of an LWW-Register CRDT for property fields (total order via Compare).
type HLC struct {
	Physical int64  `json:"wall"`     // wall clock ms
	Logical  uint32 `json:"logical"`  // logical counter
	NodeID   string `json:"node"`     // stable replica id (client tab); tie-breaker
}

// ZeroHLC is the minimal element (no event yet).
var ZeroHLC = HLC{}

// Compare returns +1 if a > b, -1 if a < b, 0 if equal (including node).
func (a HLC) Compare(b HLC) int {
	if a.Physical != b.Physical {
		if a.Physical > b.Physical {
			return 1
		}
		return -1
	}
	if a.Logical != b.Logical {
		if a.Logical > b.Logical {
			return 1
		}
		return -1
	}
	return strings.Compare(a.NodeID, b.NodeID)
}

// After is true iff a is strictly greater than b in the total order.
func (a HLC) After(b HLC) bool {
	return a.Compare(b) > 0
}

// IsValidClientHLC rejects empty node and absurd sizes (DoS).
func (h HLC) IsValidClientHLC() bool {
	if h.NodeID == "" || len(h.NodeID) > 64 {
		return false
	}
	if h.Physical < 0 {
		return false
	}
	return true
}

// Tick generates the next local event time (client calls before sending a mutation).
func (h HLC) Tick(nowMs int64) HLC {
	out := h
	if nowMs > out.Physical {
		out.Physical = nowMs
		out.Logical = 0
	} else {
		out.Logical++
	}
	return out
}

// Receive merges a remote timestamp into local state (server or client on fan-out).
// Implements the classic HLC receive step so future local ticks are causally after `remote`.
func (h *HLC) Receive(nowMs int64, remote HLC) {
	if remote.Physical > h.Physical {
		h.Physical = remote.Physical
		h.Logical = remote.Logical + 1
		return
	}
	if remote.Physical == h.Physical && remote.Logical >= h.Logical {
		h.Logical = remote.Logical + 1
		return
	}
	// remote < local on (physical, logical) — move wall forward
	if nowMs > h.Physical {
		h.Physical = nowMs
		h.Logical = 0
		return
	}
	if nowMs == h.Physical {
		h.Logical++
		return
	}
	// nowMs < h.Physical (skew): keep physical, bump logical
	h.Logical++
}

// Max returns the lexicographic maximum (latest event in LWW sense).
func MaxHLC(a, b HLC) HLC {
	if a.Compare(b) >= 0 {
		return a
	}
	return b
}

func hlcFromJSON(raw []byte) (HLC, error) {
	var h HLC
	if len(raw) == 0 {
		return ZeroHLC, nil
	}
	if err := json.Unmarshal(raw, &h); err != nil {
		return ZeroHLC, err
	}
	return h, nil
}

func hlcToJSON(h HLC) ([]byte, error) {
	return json.Marshal(h)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
