package config

import (
	"strings"
	"unicode"
)

// NormalizeServiceName converts an arbitrary name into a python-safe snake_case identifier.
// Rules:
// - lowercases
// - converts '-' and whitespace to '_'
// - converts any non [a-z0-9_] to '_'
// - collapses repeated '_' and trims leading/trailing '_'
// - prefixes with "svc_" if the name would start with a digit
func NormalizeServiceName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "service"
	}

	var b strings.Builder
	b.Grow(len(raw))

	lastUnderscore := false
	for _, r := range raw {
		if r == '-' || unicode.IsSpace(r) {
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
			continue
		}

		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			if r == '_' {
				if lastUnderscore {
					continue
				}
				lastUnderscore = true
			} else {
				lastUnderscore = false
			}
			b.WriteRune(r)
			continue
		}

		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}

	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "service"
	}
	if name[0] >= '0' && name[0] <= '9' {
		return "svc_" + name
	}
	return name
}

func NormalizeEnvVarName(raw string) string {
	return strings.ToUpper(NormalizeServiceName(raw))
}
