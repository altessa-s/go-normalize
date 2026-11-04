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
| `collapse_spaces`       | `string`, pointers to `string`      | Replaces each run of white space with one ASCII space; does not trim    |
| `truncate(max=N)`       | `string`, pointers to `string`      | Keeps at most N runes, never splitting a character                      |
| `digits_only`           | `string`, pointers to `string`      | Keeps the ASCII digits 0-9                                              |
| `email`                 | `string`, pointers to `string`      | Trims and lower-cases the domain after the last `@`; does not validate  |
| `nil_on_zero`           | pointers of any type                | Sets the innermost pointer to nil when it points to a zero value        |
| `empty_on_nil`          | pointers of any type                | Points nil pointers, through the chain, at new zero values              |
| `dedupe`                | slices of strings, numbers, bools   | Removes repeats and keeps first occurrences; other slices are unchanged |
| `utc`                   | `time.Time`, pointers to it         | Converts to UTC; the instant is unchanged                               |
| `truncate_time(unit=U)` | `time.Time`, pointers to it         | Rounds down to a multiple of U (`1s`, `500ms`, or a bare unit like `us`) |
| `clamp(min=A;max=B)`    | integers and floats                 | Limits to [A, B]; either bound may be omitted; NaN is left unchanged    |
| `url`                   | `string`, pointers to `string`      | Lower-cases scheme and host, drops default ports, empty path becomes `/` |

Named string types keep their type. `url` keeps query and fragment and only trims relative, opaque or unparsable URLs. `remove_bad_symbols` leaves a
nonempty string unchanged when nothing would remain. A missing or invalid `max` makes `truncate` fail with `ErrInvalidParameter`, as do a missing or
non-positive `unit` for `truncate_time` and `clamp` bounds that are absent, not numbers, outside the field type's range or out of order.

## Key types

| Identifier            | Description                                                                                         |
|-----------------------|-----------------------------------------------------------------------------------------------------|
| `Modifier`            | `func(reflect.Value, map[string]string) ModifierResult`                                             |
| `ModifierResult`      | The value to store and an optional `*ModifierError`                                                 |
| `ModifierError`       | Field name, path, modifier name and cause; unwraps to the cause                                     |
| `ErrInvalidParameter` | Cause of a `ModifierError` for a tag parameter the modifier cannot use                             |
| `Registry`            | Name-to-modifier table; lookups take no lock, `Register` copies the table; the zero value is empty |
| `DefaultRegistry`     | The registry holding the built-ins, used by `normalize.Normalize`                                   |
| `Parse`               | Splits a clause such as `phone(region=US; strict)` into a name and a fresh parameter map            |
| `ApplyStringModifier` | Runs a string transform on `string` and pointer-to-`string` values                                  |
| `ApplyModifier[T]`   | Runs a transform on `T` and pointers to it; predeclared `T` also matches named types of its kind    |

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
