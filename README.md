# go-normalize

```go
import "github.com/altessa-s/go-normalize"
```

Reflection-based normalization for Go structs using `normalize` tags. Built-in modifiers work without optional modules; phone normalization is
opt-in through the `phone` module.

## Usage

```go
type Contact struct {
	Name  string  `normalize:"trim,lowercase"`
	Email *string `normalize:"trim,lowercase,nil_on_empty"`
}

contact := Contact{Name: "  JANE "}
if err := normalize.Normalize(&contact); err != nil { /* handle error */ }
```

Tags list modifier clauses separated by commas and applied in order; a clause may take parameters, as in `phone(region=US)`. Traversal follows
pointers, slices, arrays, maps and interfaces, stops at cycles, and runs a `CustomNormalizer` after the tagged fields of its struct. `-` skips a
field and `custom` leaves it to the parent's `CustomNormalizer`.

## Packages

| Package                                    | Purpose                                                                   |
|--------------------------------------------|---------------------------------------------------------------------------|
| `normalize`                                | `Normalize`, `NormalizeWithRegistry`, `CustomNormalizer`, sentinel errors |
| [`modifiers`](modifiers/README.md)         | Modifier types, the concurrency-safe registry and the built-in modifiers  |
| [`normalizetest`](normalizetest/README.md) | Test helpers for consumers                                                |
| [`phone`](phone/README.md)                 | Opt-in phone-number modifier, a separate module                           |
| [`text`](text/README.md)                   | Opt-in Unicode modifiers: `nfc`, `nfkc`, `fold`, `title`, a separate module |
| [`idna`](idna/README.md)                   | Opt-in punycode conversion of domains, emails and URL hosts, a separate module |

## Key types

| Identifier              | Description                                                                                       |
|-------------------------|---------------------------------------------------------------------------------------------------|
| `Normalize`             | Applies the default registry to a non-nil pointer to a struct                                     |
| `NormalizeWithRegistry` | Applies modifiers from any `ModifierSource`, such as a `*modifiers.Registry` with `phone` added   |
| `ModifierSource`        | `Get(name) (modifiers.Modifier, bool)`; a nil source resolves no modifiers                        |
| `CustomNormalizer`      | `Normalize() error`, called after the tagged fields of the struct                                 |
| `ErrInvalidInput`       | The value is not a non-nil pointer to a struct                                                    |
| `ErrModifierNotFound`   | A tag names a modifier the source does not resolve                                                |
| `ErrNonReflexiveMapKey` | A map value changed under a key that cannot address it again, such as NaN                         |

Modifier failures are returned as `*modifiers.ModifierError` with the field name and path; map keys appear as `{key}` so their contents never
reach error messages.

## Composing registries

`phone.Register` adds the `phone` modifier to a registry. To use it with built-ins, copy the ones you need from `modifiers.DefaultRegistry()`:

```go
registry := modifiers.NewRegistry()
for _, name := range []string{"trim", "nil_on_empty"} {
	if m, ok := modifiers.DefaultRegistry().Get(name); ok {
		registry.Register(name, m)
	}
}
phone.Register(registry)
err := normalize.NormalizeWithRegistry(&contact, registry)
```
