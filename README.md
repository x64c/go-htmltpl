# htmltpl

A Go HTML template compiler with layout inheritance, partials, and load-time variables.

Built on top of Go's `html/template`. Adds `{@extend@}`, `{@include@}`, and `{@var@}` directives that resolve at load time, producing fully-built `*template.Template` values ready for `Execute()`.

## API

```go
func CompileTemplates(srcDir fs.FS) (map[string]*template.Template, error)
```

Input: any `fs.FS` (real directory, embedded FS, in-memory).
Output: map of template key to fully-built template. Each template is a Go template set with all directives resolved.

```go
tpls, err := htmltpl.CompileTemplates(os.DirFS("templates/html"))
tpls["pages/home"].Execute(w, data)
```

## File Extension

`.ghtml` — all files must use this extension.

## Template Key

Relative path from the source root, without the `.ghtml` extension:

```
templates/html/layouts/base.ghtml  →  "layouts/base"
templates/html/pages/home.ghtml    →  "pages/home"
```

## Directives

Custom directives processed at load time. They use `{@ @}` syntax, separate from Go's `{{ }}`.

### `{@extend foo/bar @}`

Declares that this file extends another template. The target and this file are compiled into a single Go template set.

- Zero or one per file, at the top
- Uses Go's template set: target parsed first, this file's `{{define}}` provides definition to target's `{{template}}`. The last definition wins.
- Multi-level chaining supported: page → child layout → root layout

Layout (`layouts/base.ghtml`):
```html
<!DOCTYPE html>
<html>
<head><title>App</title></head>
<body>
    <nav>{{block "nav" .}}default nav{{end}}</nav>
    <main>{{block "content" .}}{{end}}</main>
</body>
</html>
```

Page (`pages/home.ghtml`):
```html
{@extend layouts/base @}

{{define "nav"}}
<ul><li>Home</li><li>About</li></ul>
{{end}}

{{define "content"}}
<h1>{{.Title}}</h1>
{{end}}
```

### `{@include foo/bar @}`

Pastes the referenced file's source text at the directive location. Recursive — included files can include other files.

- Target file must not have `{@extend@}`
- Circular includes detected and reported with full chain

```html
{{define "nav"}}
{@include partials/main_nav @}
{{end}}
```

### `{@var Name @}`

Replaced with the value of `Name` from `.vars.json` in the source root directory. Load-time variable substitution — values are baked into the template source.

`.vars.json`:
```json
{
    "AssetVersion": "20260329"
}
```

Template:
```html
<script src="/static/js/app.mjs?v={@var AssetVersion @}"></script>
```

Output:
```html
<script src="/static/js/app.mjs?v=20260329"></script>
```

- Error if a `{@var@}` references a name not in `.vars.json`
- If `.vars.json` doesn't exist, `{@var@}` directives will error

## Escaping

To output a literal `{@` in the rendered HTML, use `{@@`:

```html
<p>Use {@@extend layout @} to extend a layout.</p>
<!-- outputs: Use {@extend layout @} to extend a layout. -->
```

## Compact / Seal — Template Flattening

Post-compilation step. Inlines all `{{template}}`/`{{block}}` calls into a single-participant template.

```go
// Compact — best effort. Skips unresolvable slots.
compacted, err := htmltpl.Compact(tpls["pages/home"])

// Seal — strict. Errors if any slot can't be resolved.
sealed, err := htmltpl.Seal(tpls["pages/home"])

// CompactAll / SealAll — run on entire map. All-or-nothing.
err := htmltpl.CompactAll(tpls)
err := htmltpl.SealAll(tpls)
```

Data argument handling:
- `nil`/omitted/`.` — inline as-is
- `.Foo`, `.Foo.Bar` — inline with dot prepending (scope-aware)
- `.Method "arg"` — inline with paren-wrapping: `(.User.Method "arg").Field`
- Literals (`"hello"`, `42`) — can't inline. Error if participant accesses fields.
- Complex (`$var`, pipes) — can't inline. Compact skips, Seal errors.

Scope-aware: `{{with}}`/`{{range}}` arguments are rewritten, inner content left as-is (dot is rebased).

## Rules

1. `{@extend@}` — zero or one per file, at the top
2. Multi-level extend chaining through separate files
3. `{@include@}` target must not have `{@extend@}`
4. No circular includes
5. `{{define}}` name collisions from `{@include@}` — last definition wins (developer's responsibility)
6. Hidden files/directories (`.` prefix) are skipped
7. Files must be valid UTF-8
