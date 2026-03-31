package htmltpl

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	fileSuffix      = ".ghtml"
	directiveEscape = "{@@"
)

// CompileTemplates walks srcDir, reads all .ghtml files, resolves directives
// ({@extend@}, {@include@}, {@var@}), and returns a map of fully-built templates.
func CompileTemplates(srcDir fs.FS) (map[string]*template.Template, error) {
	// Phase 1: Load all .ghtml files into source map
	sources, err := loadSources(srcDir)
	if err != nil {
		return nil, err
	}

	// Phase 1b: Load .vars.json if present
	vars, err := loadVars(srcDir)
	if err != nil {
		return nil, err
	}

	// Phase 1c: Resolve @var directives in all sources
	for key, source := range sources {
		resolved, err := resolveVars(key, source, vars)
		if err != nil {
			return nil, err
		}
		sources[key] = resolved
	}

	// Phase 2: Process each file → resolve directives → parse → output
	templates := make(map[string]*template.Template, len(sources))

	for key, source := range sources {
		// Resolve @include (text replacement)
		resolved, err := resolveIncludes(key, source, sources, nil)
		if err != nil {
			return nil, err
		}

		// Check for @extend
		extendKey, afterExtend, err := parseExtend(key, resolved)
		if err != nil {
			return nil, err
		}

		if extendKey == "" {
			// No extend — parse as standalone single-participant arena
			cleaned := cleanEscapes(resolved)
			t, err := template.New(key).Parse(cleaned)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			templates[key] = t
			continue
		}

		// Has extend — build arena: layout first, page content after
		layoutSource, ok := sources[extendKey]
		if !ok {
			return nil, fmt.Errorf("%s: @extend target not found: %s", key, extendKey)
		}

		// Resolve includes in the layout too
		resolvedLayout, err := resolveIncludes(extendKey, layoutSource, sources, nil)
		if err != nil {
			return nil, err
		}

		// Follow the extend chain (layout may extend another layout)
		chain, err := buildExtendChain(extendKey, resolvedLayout, sources)
		if err != nil {
			return nil, err
		}

		// Build arena: root layout first, then each child, page last
		rootKey := chain[0].key
		t, err := template.New(rootKey).Parse(cleanEscapes(chain[0].source))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rootKey, err)
		}

		// Add each child layout
		for i := 1; i < len(chain); i++ {
			_, err = t.New(chain[i].key).Parse(cleanEscapes(chain[i].source))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", chain[i].key, err)
			}
		}

		// Add the page itself (afterExtend has the {{define}} blocks)
		_, err = t.New(key).Parse(cleanEscapes(afterExtend))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}

		templates[key] = t
	}

	return templates, nil
}

// cleanEscapes replaces {@@ with {@ after all directives are resolved.
func cleanEscapes(source string) string {
	return strings.ReplaceAll(source, directiveEscape, directiveOpen)
}

type chainEntry struct {
	key    string
	source string
}

// buildExtendChain follows the @extend chain from a layout up to the root.
// Returns entries in order: root first, deepest child last.
func buildExtendChain(key string, source string, sources map[string]string) ([]chainEntry, error) {
	var chain []chainEntry
	visited := map[string]bool{}

	currentKey := key
	currentSource := source

	for {
		if visited[currentKey] {
			return nil, fmt.Errorf("circular @extend: %s", currentKey)
		}
		visited[currentKey] = true

		extendKey, afterExtend, err := parseExtend(currentKey, currentSource)
		if err != nil {
			return nil, err
		}

		if extendKey == "" {
			// Root layout — no extend
			chain = append(chain, chainEntry{currentKey, currentSource})
			break
		}

		// This layout extends another — store the afterExtend part
		chain = append(chain, chainEntry{currentKey, afterExtend})

		// Follow the chain
		parentSource, ok := sources[extendKey]
		if !ok {
			return nil, fmt.Errorf("%s: @extend target not found: %s", currentKey, extendKey)
		}

		// Resolve includes in parent
		resolvedParent, err := resolveIncludes(extendKey, parentSource, sources, nil)
		if err != nil {
			return nil, err
		}

		currentKey = extendKey
		currentSource = resolvedParent
	}

	// Reverse: root first, deepest child last
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}

// loadSources walks srcDir and reads all .ghtml files into a map keyed by
// relative path without extension.
func loadSources(srcDir fs.FS) (map[string]string, error) {
	sources := make(map[string]string)

	err := fs.WalkDir(srcDir, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		// Skip hidden dirs/files (but not the root ".")
		if name != "." && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, fileSuffix) {
			return nil
		}

		fileBytes, err := fs.ReadFile(srcDir, path)
		if err != nil {
			return err
		}
		if !utf8.Valid(fileBytes) {
			return fmt.Errorf("file %s is not valid UTF-8", path)
		}

		key := strings.TrimSuffix(filepath.ToSlash(path), fileSuffix)
		if _, exists := sources[key]; exists {
			return fmt.Errorf("duplicate template key: %s", key)
		}

		sources[key] = string(fileBytes)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sources, nil
}
