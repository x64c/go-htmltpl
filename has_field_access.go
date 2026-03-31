package htmltpl

import "strings"

// hasFieldAccess reports whether content contains dot field/method access
// inside {{ }} actions. e.g. {{.Name}}, {{.Foo.Bar}}.
// Bare {{.}} does not count.
func hasFieldAccess(content string) bool {
	remaining := content
	for {
		// Find next {{ action
		start := strings.Index(remaining, "{{")
		if start == -1 {
			return false // no more actions
		}
		// Find matching }}
		end := strings.Index(remaining[start:], "}}")
		if end == -1 {
			return false // unclosed action, give up
		}
		// Extract action content between {{ and }}
		action := remaining[start+2 : start+end]
		// Scan action for dot followed by field start (letter or _)
		for i := 0; i < len(action)-1; i++ {
			if action[i] == '.' && isFieldStart(action[i+1]) {
				return true // found field access like .Name
			}
		}
		// Move past this action
		remaining = remaining[start+end+2:]
	}
}

func isFieldStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}
