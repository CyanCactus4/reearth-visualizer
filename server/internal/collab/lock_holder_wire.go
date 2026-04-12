package collab

import (
	"strings"
	"unicode/utf8"
)

const lockHolderSep = "\x1f"

// LockHolderWire is the value stored for a collab resource lock (bare userId when clientID is empty).
func LockHolderWire(userID, clientID string) string {
	if clientID == "" {
		return userID
	}
	return userID + lockHolderSep + clientID
}

// ParseLockHolderWire splits a stored lock holder into user id and optional tab client id.
func ParseLockHolderWire(s string) (userID, clientID string) {
	i := strings.Index(s, lockHolderSep)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+len(lockHolderSep):]
}

// LockHeldBySameTab is true when this WebSocket tab holds the lock (same user + same client id, or legacy bare user id with no client id).
func LockHeldBySameTab(holder string, userID, clientID string) bool {
	if holder == LockHolderWire(userID, clientID) {
		return true
	}
	if clientID == "" && holder == userID {
		return true
	}
	return false
}

// HTTPLockBlocksUser is used for requests without a tab id: blocks if another user holds the lock,
// or if the lock is tied to a specific tab (wired holder).
func HTTPLockBlocksUser(holder, uid string) bool {
	if holder == "" {
		return false
	}
	hu, hc := ParseLockHolderWire(holder)
	if hu != uid {
		return true
	}
	return hc != ""
}

// NormalizeCollabClientID returns a safe tab id from the WebSocket query, or "" if invalid.
func NormalizeCollabClientID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 128 {
		return ""
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return ""
		}
	}
	return s
}
