# modifiers

```go
import "github.com/altessa-s/go-normalize/modifiers"
```

Modifier types, a concurrency-safe registry and the built-in string modifiers used by [`normalize`](../README.md) tags.

## Built-in modifiers

| Name                    | Applies to                          | Effect                                                                  |
|-------------------------|-------------------------------------|-------------------------------------------------------------------------|
| `trim`                  | `string`, pointers to `string`      | Removes leading and trailing white space                                |
| `lowercase`             | `string`, pointers to `string`      | Converts to lower case                                                  |
| `uppercase`             | `string`, pointers to `string`      | Converts to upper case                                                  |
| `nil_on_empty`          | pointers to `string`                | Sets the innermost pointer to nil when the string is empty              |
| `remove_bad_symbols`    | `string`, pointers to `string`      | Removes control and directional formatting characters                   |
| `remove_empty_elements` | `[]string`, `[]*string`             | Drops nil and blank elements                                            |

Named string types keep their type. `remove_bad_symbols` leaves a nonempty string unchanged when nothing would remain.

## Key types

| Identifier            | Description                                                                                         |
|-----------------------|-----------------------------------------------------------------------------------------------------|
| `Modifier`            | `func(reflect.Value, map[string]string) ModifierResult`                                             |
| `ModifierResult`      | The value to store and an optional `*ModifierError`                                                 |
| `ModifierError`       | Field name, path, modifier name and cause; unwraps to the cause                                     |
| `Registry`            | Name-to-modifier table; lookups take no lock, `Register` copies the table; the zero value is empty |
| `DefaultRegistry`     | The registry holding the built-ins, used by `normalize.Normalize`                                   |
| `Parse`               | Splits a clause such as `phone(region=US; strict)` into a name and a fresh parameter map            |
| `ApplyStringModifier` | Runs a string transform on `string` and pointer-to-`string` values                                  |

## Usage

```go
registry := modifiers.NewRegistry()
registry.Register("slug", func(v reflect.Value, params map[string]string) modifiers.ModifierResult {
	return modifiers.ApplyStringModifier(v, func(s string) (string, bool) {
		out := strings.Join(strings.Fields(s), params["sep"])
		return out, out != s
	})
})
```

Register modifiers during setup: each `Register` call copies the table so that lookups never block.
