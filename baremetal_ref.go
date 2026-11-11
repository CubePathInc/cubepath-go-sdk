package cubepath

import (
	"fmt"
	"strings"
)

// A baremetal server is named by a reference: its number (digits only, the "id" of the
// API) or its uuid (8-4-4-4-12 hexadecimal, any case). The routes of the API take both.
// These helpers are specific to baremetal: VPS and networks keep integer ids.

// maxBaremetalNumber is the largest number the API accepts (BIGINT max, 2^63-1).
const maxBaremetalNumber = "9223372036854775807"

// IsBaremetalRef reports whether ref has the shape of a baremetal reference: a number in
// the range the API accepts (1 to 2^63-1, leading zeros ignored) or a uuid.
func IsBaremetalRef(ref string) bool {
	return isBaremetalNumber(ref) || isUUID(ref)
}

func isBaremetalNumber(s string) bool {
	if !isDigits(s) {
		return false
	}
	t := strings.TrimLeft(s, "0")
	if t == "" || len(t) > len(maxBaremetalNumber) {
		return false
	}
	return len(t) < len(maxBaremetalNumber) || t <= maxBaremetalNumber
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

// validateBaremetalRef rejects, before any request, a reference that is neither a number
// nor a uuid (an empty string, a path, a number with a sign or a space...).
func validateBaremetalRef(ref string) error {
	if !IsBaremetalRef(ref) {
		return fmt.Errorf("invalid baremetal reference %q: expected a number (1 to 9223372036854775807) or a uuid", ref)
	}
	return nil
}

// normalizeBaremetalRef lower-cases a uuid and leaves a number as given.
func normalizeBaremetalRef(ref string) string {
	if isUUID(ref) {
		return strings.ToLower(ref)
	}
	return ref
}

// sameBaremetalRef reports whether ref (already validated and normalized) names a server
// with the given number and uuid. A number is compared without leading zeros, as the API
// does; uuid is empty when the API does not send one.
func sameBaremetalRef(ref string, id int, uuid string) bool {
	if isUUID(ref) {
		return uuid != "" && strings.EqualFold(ref, uuid)
	}
	return strings.TrimLeft(ref, "0") == fmt.Sprint(id)
}
