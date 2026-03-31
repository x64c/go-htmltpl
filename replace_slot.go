package htmltpl

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

// Matches {{template "name" data}} and {{block "name" data}}
// group 1: name, group 2: data arg (may be empty)
var reSlotCall = regexp.MustCompile(`\{\{(?:template|block) "([^"]+)"(?: ([^}]+))?}}`)

// replaceSlot replaces {{template}}/{{block}} calls with inlined participant content.
// parentPrefix: the accumulated dot context from parent (starts as ".").
// strict: true = error on unresolved slot, false = skip unresolved.
// Returns: new source, whether any replacement happened, error.
func replaceSlot(source string, t *template.Template, parentPrefix string, strict bool) (string, bool, error) {
	replaced := false
	var retErr error

	result := reSlotCall.ReplaceAllStringFunc(source, func(match string) string {
		// Stop processing if a previous match already errored
		if retErr != nil {
			return match
		}

		// Extract name and data arg from the match
		groups := reSlotCall.FindStringSubmatch(match)
		name := groups[1]
		dataArg := strings.TrimSpace(groups[2])

		// Lookup participant in the arena
		tmpl := t.Lookup(name)
		if tmpl == nil {
			if strict {
				retErr = fmt.Errorf("unresolved slot: %q", name)
			}
			return match // not found — leave as-is
		}

		// Get participant's content
		content := tmpl.Tree.Root.String()

		// Classify the data argument
		kind := classifyDataArg(dataArg)

		prefixForChildren := parentPrefix

		switch kind {
		case dataArgLiteral:
			// Literal data — can't inline
			if hasFieldAccess(content) {
				// Guaranteed runtime crash — always error
				retErr = fmt.Errorf("slot %q receives literal %s but accesses fields", name, dataArg)
			} else if strict {
				// Seal mode — can't leave any slot unresolved
				retErr = fmt.Errorf("slot %q: can't inline literal data %s", name, dataArg)
			}
			return match

		case dataArgOther:
			// $var, pipe, func call — can't flatten
			if strict {
				retErr = fmt.Errorf("slot %q: can't inline complex data arg %s", name, dataArg)
			}
			return match

		case dataArgDotPath:
			// .Foo, .Foo.Bar — accumulate prefix
			if parentPrefix == "." {
				prefixForChildren = dataArg
			} else {
				prefixForChildren = parentPrefix + dataArg
			}

		case dataArgMethodWithArgs:
			// .Method "arg" — wrap in parens: (.User.Method "arg")
			if parentPrefix == "." {
				prefixForChildren = "(" + dataArg + ")"
			} else {
				prefixForChildren = "(" + parentPrefix + dataArg + ")"
			}

		default:
			// dataArgNil — prefixForChildren stays as parentPrefix
		}

		// Rewrite dot references if prefix is not bare dot
		if prefixForChildren != "." {
			content = rewriteAllActions(content, prefixForChildren)
		}

		replaced = true
		return content
	})

	return result, replaced, retErr
}
