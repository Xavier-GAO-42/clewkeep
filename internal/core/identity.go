package core

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	RecordKindSession  = "session"
	RecordKindSubagent = "subagent"
)

func ValidateNativeID(field, value string) error {
	if value == "" {
		return fmt.Errorf("missing %s", field)
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("invalid %s: leading or trailing whitespace", field)
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 512 {
		return fmt.Errorf("invalid %s: must be valid UTF-8 and at most 512 characters", field)
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == '/' || r == '\\' {
			return fmt.Errorf("invalid %s: contains a control character or path separator", field)
		}
	}
	return nil
}

func CodexCanonicalID(nativeSessionID string) (string, error) {
	if err := ValidateNativeID("Codex thread id", nativeSessionID); err != nil {
		return "", err
	}
	return "codex/" + nativeSessionID, nil
}

func ClaudeCanonicalID(nativeSessionID, nativeAgentID string) (string, error) {
	if err := ValidateNativeID("Claude sessionId", nativeSessionID); err != nil {
		return "", err
	}
	if nativeAgentID == "" {
		return "claude/" + nativeSessionID, nil
	}
	if err := ValidateNativeID("Claude agentId", nativeAgentID); err != nil {
		return "", err
	}
	return "claude/" + nativeSessionID + "/agent/" + nativeAgentID, nil
}

func ValidateCanonicalID(id string) error {
	if native, ok := strings.CutPrefix(id, "codex/"); ok {
		return ValidateNativeID("Codex thread id", native)
	}
	if native, ok := strings.CutPrefix(id, "claude/"); ok {
		sessionID, agentID, hasAgent := strings.Cut(native, "/agent/")
		if err := ValidateNativeID("Claude sessionId", sessionID); err != nil {
			return err
		}
		if !hasAgent {
			return nil
		}
		return ValidateNativeID("Claude agentId", agentID)
	}
	return fmt.Errorf("canonical id %q has an unsupported provider namespace", id)
}

func ValidateThreadIdentity(thread Thread) error {
	var expected string
	var err error
	switch thread.Provider {
	case "codex":
		if thread.RecordKind != RecordKindSession || thread.NativeAgentID != "" {
			return fmt.Errorf("Codex record must be a session without native_agent_id")
		}
		expected, err = CodexCanonicalID(thread.NativeSessionID)
	case "claude-code":
		switch thread.RecordKind {
		case RecordKindSession:
			if thread.NativeAgentID != "" {
				return fmt.Errorf("Claude session cannot have native_agent_id")
			}
		case RecordKindSubagent:
			if thread.NativeAgentID == "" {
				return fmt.Errorf("Claude subagent is missing native_agent_id")
			}
		default:
			return fmt.Errorf("invalid Claude record_kind %q", thread.RecordKind)
		}
		expected, err = ClaudeCanonicalID(thread.NativeSessionID, thread.NativeAgentID)
	default:
		return fmt.Errorf("unsupported provider %q", thread.Provider)
	}
	if err != nil {
		return err
	}
	if thread.ID != expected {
		return fmt.Errorf("canonical id %q does not match identity; want %q", thread.ID, expected)
	}
	return nil
}

// ValidateThreadSet checks the identity contract for an addressable collection.
// Canonical IDs are map keys throughout ctx, so duplicates must fail before a
// reader can silently overwrite one record with another.
func ValidateThreadSet(threads []Thread) error {
	seen := make(map[string]bool, len(threads))
	for index, thread := range threads {
		if err := ValidateThreadIdentity(thread); err != nil {
			return fmt.Errorf("record %d with id %q has invalid identity: %w", index, thread.ID, err)
		}
		if seen[thread.ID] {
			return fmt.Errorf("duplicate canonical id %q", thread.ID)
		}
		seen[thread.ID] = true
	}
	return nil
}
