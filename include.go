package htmltpl

import (
	"fmt"
	"strings"
)

// resolveIncludes replaces all {@include path @} directives with the referenced
// file's source text from the sources map. Resolves recursively.
// visited tracks the include chain to detect circular includes.
func resolveIncludes(key string, source string, sources map[string]string, visited []string) (string, error) {
	// Check circular
	for _, v := range visited {
		if v == key {
			return "", fmt.Errorf("circular include: %s", strings.Join(append(visited, key), " → "))
		}
	}
	visited = append(visited, key)

	var result strings.Builder
	remaining := source

	for {
		openIdx := strings.Index(remaining, directiveOpen)
		if openIdx == -1 {
			result.WriteString(remaining)
			break
		}

		closeIdx := strings.Index(remaining[openIdx:], directiveClose)
		if closeIdx == -1 {
			return "", fmt.Errorf("%s: unclosed directive: no matching %s", key, directiveClose)
		}
		closeIdx += openIdx

		inner := strings.TrimSpace(remaining[openIdx+len(directiveOpen) : closeIdx])

		if !strings.HasPrefix(inner, "include ") {
			// Not an include directive — keep it as-is
			result.WriteString(remaining[:closeIdx+len(directiveClose)])
			remaining = remaining[closeIdx+len(directiveClose):]
			continue
		}

		includeKey := strings.TrimSpace(inner[len("include "):])
		if includeKey == "" {
			return "", fmt.Errorf("%s: empty @include target", key)
		}

		includeSrc, ok := sources[includeKey]
		if !ok {
			return "", fmt.Errorf("%s: @include target not found: %s", key, includeKey)
		}

		// Check that included file doesn't have @extend
		extendKey, _, err := parseExtend(includeKey, includeSrc)
		if err != nil {
			return "", err
		}
		if extendKey != "" {
			return "", fmt.Errorf("%s: @include target %s has @extend (not allowed)", key, includeKey)
		}

		// Recursively resolve includes in the included source
		resolved, err := resolveIncludes(includeKey, includeSrc, sources, visited)
		if err != nil {
			return "", err
		}

		// Write everything before the directive + the resolved include
		result.WriteString(remaining[:openIdx])
		result.WriteString(resolved)
		remaining = remaining[closeIdx+len(directiveClose):]
	}

	return result.String(), nil
}
