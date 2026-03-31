package htmltpl

import (
	"fmt"
	"strings"
)

const (
	directiveOpen  = "{@"
	directiveClose = "@}"
)

// parseExtend looks for {@extend layout/name @} at the top of the source.
// Returns the layout key and the remaining source after the directive.
// If no @extend found, returns ("", original source, nil).
func parseExtend(key string, source string) (string, string, error) {
	trimmed := strings.TrimLeft(source, " \t\n\r")

	if !strings.HasPrefix(trimmed, directiveOpen) {
		return "", source, nil
	}

	closeIdx := strings.Index(trimmed, directiveClose)
	if closeIdx == -1 {
		return "", "", fmt.Errorf("%s: unclosed directive: no matching %s", key, directiveClose)
	}

	inner := strings.TrimSpace(trimmed[len(directiveOpen):closeIdx])

	if !strings.HasPrefix(inner, "extend ") {
		return "", source, nil
	}

	layoutKey := strings.TrimSpace(inner[len("extend "):])
	if layoutKey == "" {
		return "", "", fmt.Errorf("%s: empty @extend target", key)
	}

	after := trimmed[closeIdx+len(directiveClose):]
	return layoutKey, after, nil
}
