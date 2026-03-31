package htmltpl

import "strings"

// rewriteAction takes a single {{ }} action INCLUDING the {{ }} braces.
// Finds dot tokens inside and prepends prefix to dot references.
// Number literals like .4 are left as-is.
// Input: "{{if .Active}}", prefix ".User" → "{{if .User.Active}}"
// Input: "{{.Name}}", prefix ".User" → "{{.User.Name}}"
// Input: "{{.}}", prefix ".User" → "{{.User}}"
// Input: "{{len .4}}", prefix ".User" → "{{len .4}}" (unchanged, .4 is number)
func rewriteAction(action string, prefix string) string {
	if prefix == "." {
		return action
	}
	var sb strings.Builder
	i := 0
	for i < len(action) {
		if action[i] != '.' {
			sb.WriteByte(action[i])
			i++
			continue
		}
		// Found a dot — extract the full dot token (.Name, .Foo.Bar, bare .)
		end := i + 1
		for end < len(action) {
			if action[end] == '.' {
				// Chained access like .Foo.Bar
				end++
				continue
			}
			if isFieldChar(action[end]) {
				end++
				continue
			}
			break
		}
		token := action[i:end]
		if !isDotRef(token) {
			// Number literal like .4 — leave as-is
			sb.WriteString(token)
			i = end
			continue
		}
		// Dot reference — prepend prefix
		if len(token) > 1 {
			// .Name → .User.Name
			sb.WriteString(prefix + token)
		} else {
			// bare . → .User
			sb.WriteString(prefix)
		}
		i = end
	}
	return sb.String()
}

func isFieldChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}
