package htmltpl

import (
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"strings"
)

const varsFile = ".vars.json"

// loadVars reads .vars.json from the fs root if it exists.
// Returns nil if the file doesn't exist.
func loadVars(srcDir fs.FS) (map[string]string, error) {
	data, err := fs.ReadFile(srcDir, varsFile)
	if err != nil {
		return nil, nil
	}

	var vars map[string]string
	if err := json.Unmarshal(data, &vars); err != nil {
		return nil, fmt.Errorf("%s: %w", varsFile, err)
	}
	return vars, nil
}

// resolveVars replaces all {@var Name @} directives with values from the vars map.
func resolveVars(key string, source string, vars map[string]string) (string, error) {
	if len(vars) == 0 {
		return source, nil
	}

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
			result.WriteString(remaining)
			break
		}
		closeIdx += openIdx

		inner := strings.TrimSpace(remaining[openIdx+len(directiveOpen) : closeIdx])

		if !strings.HasPrefix(inner, "var ") {
			result.WriteString(remaining[:closeIdx+len(directiveClose)])
			remaining = remaining[closeIdx+len(directiveClose):]
			continue
		}

		varName := strings.TrimSpace(inner[len("var "):])
		if varName == "" {
			return "", fmt.Errorf("%s: empty @var name", key)
		}

		val, ok := vars[varName]
		if !ok {
			return "", fmt.Errorf("%s: @var %q not found in %s", key, varName, varsFile)
		}

		result.WriteString(remaining[:openIdx])
		result.WriteString(val)
		remaining = remaining[closeIdx+len(directiveClose):]
	}

	return result.String(), nil
}
