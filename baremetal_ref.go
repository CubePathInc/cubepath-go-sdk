package cubepath

import (
	"fmt"
	"strings"
)

// A baremetal server is named by its uuid (8-4-4-4-12 hexadecimal, any case, sent in lower
// case), which is also its "id" in the API. These helpers are specific to baremetal: VPS and
// networks keep integer ids.

// IsBaremetalRef reports whether ref has the shape of a baremetal reference: a uuid.
func IsBaremetalRef(ref string) bool {
	return isUUID(ref)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// validateBaremetalRef rejects, before any request, a reference that is not a uuid (an empty
// string, a number, a path...).
func validateBaremetalRef(ref string) error {
	if !IsBaremetalRef(ref) {
		return fmt.Errorf("invalid baremetal reference %q: baremetal reference must be a uuid", ref)
	}
	return nil
}

// normalizeBaremetalRef lower-cases a uuid.
func normalizeBaremetalRef(ref string) string {
	return strings.ToLower(ref)
}

// baremetalRef validates ref and returns it in lower case.
func baremetalRef(ref string) (string, error) {
	if err := validateBaremetalRef(ref); err != nil {
		return "", err
	}
	return normalizeBaremetalRef(ref), nil
}

// baremetalEcho validates the target of another resource (route next hop, BGP peer, load
// balancer target, alert target) when it names a baremetal server, and returns it in lower
// case. Targets of other types are returned as given.
func baremetalEcho(targetType, target string) (string, error) {
	if targetType != "baremetal" {
		return target, nil
	}
	return baremetalRef(target)
}
