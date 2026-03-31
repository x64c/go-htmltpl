package htmltpl

import "strings"

// rewriteAllActions walks the full content of a participant (HTML + {{ }} actions mixed).
// Finds each {{ }} action and rewrites dot references by calling rewriteAction() with prefix.
// Scope-aware: tracks {{with}}/{{range}} depth. Only rewrites at depth 0.
// Inner content of {{with}}/{{range}} blocks is left as-is (dot is rebased inside).
// The opening {{with .X}}/{{range .X}} argument IS rewritten at depth 0.
// Input: "<div>{{.Name}}</div>{{with .Profile}}{{.City}}{{end}}", prefix ".User"
// Output: "<div>{{.User.Name}}</div>{{with .User.Profile}}{{.City}}{{end}}"
func rewriteAllActions(content string, prefix string) string {
	var sb strings.Builder
	i := 0
	depth := 0

	for i < len(content) {
		// Not an action start — copy as-is
		if !strings.HasPrefix(content[i:], "{{") {
			sb.WriteByte(content[i])
			i++
			continue
		}

		// Find matching }}
		end := strings.Index(content[i:], "}}")
		if end == -1 {
			// No closing — write rest as-is
			sb.WriteString(content[i:])
			break
		}
		end += i + 2 // position after }}

		// Full action including {{ }}
		action := content[i:end]
		// Inner content between {{ and }}
		inner := strings.TrimSpace(content[i+2 : end-2])

		// {{end}} — decrease depth
		if inner == "end" {
			sb.WriteString(action)
			i = end
			if depth > 0 {
				depth--
			}
			continue
		}

		// {{with ...}} or {{range ...}} — scope-resetting block
		if strings.HasPrefix(inner, "with ") || strings.HasPrefix(inner, "range ") {
			if depth == 0 {
				// Rewrite the argument dots
				sb.WriteString(rewriteAction(action, prefix))
			} else {
				// Already inside a scope block — don't rewrite
				sb.WriteString(action)
			}
			i = end
			depth++
			continue
		}

		// Regular action — rewrite if at top level
		if depth == 0 {
			sb.WriteString(rewriteAction(action, prefix))
		} else {
			// Inside with/range — dot is rebased, don't rewrite
			sb.WriteString(action)
		}
		i = end
	}

	return sb.String()
}
