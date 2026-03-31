package htmltpl

import (
	"fmt"
	"html/template"
)

// Compact flattens a compiled template into a single participant by inlining
// {{template}}/{{block}} calls. Unresolved slots (participant not found) are
// left as-is. Errors on guaranteed runtime crashes (literal data + field access).
// Returns a new *template.Template.
func Compact(t *template.Template) (*template.Template, error) {
	source := t.Tree.Root.String()

	// Loop until no more replacements
	for {
		result, replaced, err := replaceSlot(source, t, ".", false)
		if err != nil {
			return nil, err
		}
		source = result
		if !replaced {
			break
		}
	}

	return template.New(t.Name()).Parse(source)
}

// CompactAll runs Compact on every compiled template in the map.
// All-or-nothing: collects results first, swaps all on success.
// Returns error on the first failure without modifying the map.
func CompactAll(compiledTpls map[string]*template.Template) error {
	results := make(map[string]*template.Template, len(compiledTpls))
	for key, t := range compiledTpls {
		compacted, err := Compact(t)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		results[key] = compacted
	}
	for key, t := range results {
		compiledTpls[key] = t
	}
	return nil
}

// SealAll runs Seal on every compiled template in the map.
// All-or-nothing: collects results first, swaps all on success.
// Returns error on the first failure without modifying the map.
func SealAll(compiledTpls map[string]*template.Template) error {
	results := make(map[string]*template.Template, len(compiledTpls))
	for key, t := range compiledTpls {
		sealed, err := Seal(t)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		results[key] = sealed
	}
	for key, t := range results {
		compiledTpls[key] = t
	}
	return nil
}

// Seal flattens a compiled template into a single participant, same as Compact,
// but errors if any {{template}}/{{block}} call cannot be resolved.
// Returns a new *template.Template with no remaining slot calls.
func Seal(t *template.Template) (*template.Template, error) {
	source := t.Tree.Root.String()

	// Loop until no more replacements
	for {
		result, replaced, err := replaceSlot(source, t, ".", true)
		if err != nil {
			return nil, err
		}
		source = result
		if !replaced {
			break
		}
	}

	return template.New(t.Name()).Parse(source)
}
