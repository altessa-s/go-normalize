# go-normalize

```go
import "github.com/altessa-s/go-normalize"
```

Reflection-based normalization for Go structs using `normalize` tags. The root module ships the built-in modifiers and has no
dependencies outside the standard library. Phone numbers (`phone`), Unicode forms and case (`text`) and internationalized domain names
(`idna`) are opt-in: each is a separate module that you add to a registry only when you need it.

## Installation

```sh
go get github.com/altessa-s/go-normalize
```

The optional modules are installed separately:

```sh
go get github.com/altessa-s/go-normalize/phone
go get github.com/altessa-s/go-normalize/text
go get github.com/altessa-s/go-normalize/idna
```

Requires Go 1.27 or later.

## Usage

```go
type Contact struct {
	Name  string  `normalize:"trim,collapse_spaces"`
	Email *string `normalize:"trim,email,nil_on_empty"`
}

contact := Contact{Name: "  Jane   Doe "}
if err := normalize.Normalize(&contact); err != nil {
	return err
}
```

Tags list modifier clauses separated by commas and applied in order; a clause may take parameters separated by semicolons, as in
`clamp(min=0;max=10)`. `-` skips a field and `custom` leaves it to the parent's `CustomNormalizer`.

Traversal visits exported fields, follows pointers, slices, arrays, maps and interfaces, descends into unexported embedded structs, and
stops at cycles. A struct whose pointer has a `Normalize() error` method is a `CustomNormalizer`: the method runs after the struct's tagged
fields.

## Built-in modifiers

`normalize.Normalize` and `modifiers.DefaultRegistry()` provide:

| Name                    | Effect                                                                    |
|-------------------------|---------------------------------------------------------------------------|
| `trim`                  | Removes leading and trailing white space                                  |
| `lowercase`             | Converts to lower case                                                    |
| `uppercase`             | Converts to upper case                                                    |
| `collapse_spaces`       | Replaces each run of white space with one ASCII space                     |
| `remove_bad_symbols`    | Removes control, bidirectional, zero width and other invisible characters |
| `truncate(max=N)`       | Keeps at most N runes                                                     |
| `digits_only`           | Keeps the ASCII digits 0-9                                                |
| `email`                 | Trims and lower-cases the domain                                          |
| `url`                   | Lower-cases scheme and host, drops default ports                          |
| `nil_on_empty`          | Sets a pointer to an empty string to nil                                  |
| `nil_on_zero`           | Sets a pointer to a zero value to nil                                     |
| `empty_on_nil`          | Points nil pointers at new zero values                                    |
| `remove_empty_elements` | Drops nil and blank elements of string slices                             |
| `dedupe`                | Removes repeated elements of slices of strings, numbers and booleans      |
| `utc`                   | Converts times to UTC                                                     |
| `truncate_time(unit=U)` | Rounds times down to a multiple of U                                      |
| `clamp(min=A;max=B)`    | Limits integers and floats to [A, B]                                      |

See [`modifiers`](modifiers/README.md) for the types each modifier applies to and its exact rules.

## Packages

| Package                                    | Purpose                                                                        |
|--------------------------------------------|--------------------------------------------------------------------------------|
| `normalize`                                | `Normalize`, `NormalizeWithRegistry`, `CustomNormalizer`, sentinel errors      |
| [`modifiers`](modifiers/README.md)         | Modifier types, the registry and the built-in modifiers                        |
| [`normalizetest`](normalizetest/README.md) | Test helpers for consumers                                                     |
| [`phone`](phone/README.md)                 | Opt-in phone-number modifier, a separate module                                |
| [`text`](text/README.md)                   | Opt-in Unicode modifiers `nfc`, `nfkc`, `fold` and `title`, a separate module  |
| [`idna`](idna/README.md)                   | Opt-in punycode conversion of domains, emails and URL hosts, a separate module |

## Key types

| Identifier              | Description                                                                        |
|-------------------------|------------------------------------------------------------------------------------|
| `Normalize`             | Applies the built-in modifiers to a non-nil pointer to a struct                    |
| `NormalizeWithRegistry` | Applies the modifiers a `ModifierSource` resolves, such as a `*modifiers.Registry` |
| `ModifierSource`        | `Get(name) (modifiers.Modifier, bool)`; a nil source resolves no modifiers         |
| `CustomNormalizer`      | `Normalize() error`, called after the tagged fields of the struct                  |
| `ErrInvalidInput`       | The value is not a non-nil pointer to a struct                                     |
| `ErrModifierNotFound`   | A tag names a modifier the source does not resolve                                 |
| `ErrResultType`         | A modifier returned a value that cannot be assigned to the field                   |
| `ErrNonReflexiveMapKey` | A map value changed under a key that cannot address it again, such as NaN          |

Modifier failures are returned as `*modifiers.Error` with the modifier name, the field name and its path; map keys appear as `{key}` so
their contents never reach error messages.

## Composing registries

`modifiers.DefaultRegistry()` returns a new registry holding the built-ins; each call returns a separate registry owned by the caller, so
registering into it changes nothing else. Add the optional modules to it:

```go
registry := modifiers.DefaultRegistry()
if err := phone.Register(registry, "US"); err != nil {
	return err
}
if err := text.Register(registry); err != nil {
	return err
}
err := normalize.NormalizeWithRegistry(&contact, registry)
```

Start from `modifiers.NewRegistry()` to resolve only the modifiers you register, and use `Registry.Clone` to derive registries from one
another.

## License

go-normalize is released under the [MIT License](LICENSE). The `phone` module depends on
[`github.com/nyaruka/phonenumbers`](https://github.com/nyaruka/phonenumbers), MIT licensed, which embeds phone number metadata
generated from Google's [libphonenumber](https://github.com/google/libphonenumber), published under the Apache License 2.0.
