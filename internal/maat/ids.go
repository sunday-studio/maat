package maat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

type IDPrefix string

const (
	ProjectIDPrefix IDPrefix = "P"
	GoalIDPrefix    IDPrefix = "G"
	TicketIDPrefix  IDPrefix = "T"
	EventIDPrefix   IDPrefix = "E"
)

func NewID(prefix IDPrefix, at time.Time) (string, error) {
	return NewIDWithReader(prefix, at, rand.Reader)
}

func NewIDWithReader(prefix IDPrefix, at time.Time, reader io.Reader) (string, error) {
	if err := validateIDPrefix(prefix); err != nil {
		return "", err
	}
	if at.IsZero() {
		return "", fmt.Errorf("id time is required")
	}
	var entropy [2]byte
	if _, err := io.ReadFull(reader, entropy[:]); err != nil {
		return "", fmt.Errorf("read id entropy: %w", err)
	}
	return fmt.Sprintf("%s-%s-%s", prefix, at.Format("20060102-150405"), hex.EncodeToString(entropy[:])), nil
}

func NewActorEventID(at time.Time, actor string) (string, error) {
	return NewActorEventIDWithReader(at, actor, rand.Reader)
}

func NewActorEventIDWithReader(at time.Time, actor string, reader io.Reader) (string, error) {
	if at.IsZero() {
		return "", fmt.Errorf("event id time is required")
	}
	actor = NormalizeIDPart(actor)
	if actor == "" {
		return "", fmt.Errorf("event actor is required")
	}
	var entropy [2]byte
	if _, err := io.ReadFull(reader, entropy[:]); err != nil {
		return "", fmt.Errorf("read event id entropy: %w", err)
	}
	return fmt.Sprintf("%s-%s-%s-%s", EventIDPrefix, at.Format("20060102-150405"), actor, hex.EncodeToString(entropy[:])), nil
}

func NormalizeIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func validateIDPrefix(prefix IDPrefix) error {
	switch prefix {
	case ProjectIDPrefix, GoalIDPrefix, TicketIDPrefix, EventIDPrefix:
		return nil
	default:
		return fmt.Errorf("unknown id prefix %q", prefix)
	}
}
