# phone

```go
import "github.com/altessa-s/go-normalize/phone"
```

Opt-in phone-number modifier for [`normalize`](../README.md) tags, in its own module so that `github.com/nyaruka/phonenumbers` stays out of
the root module's requirements.

## Usage

```go
registry := modifiers.NewRegistry()
phone.Register(registry)

type Contact struct {
	Mobile string `normalize:"phone(region=US)"`
}
err := normalize.NormalizeWithRegistry(&contact, registry)
```

## Key types

| Identifier        | Description                                                                                       |
|-------------------|---------------------------------------------------------------------------------------------------|
| `Register`        | Adds the `phone` modifier to a `Registrar`; nothing is registered implicitly                      |
| `Registrar`       | `Register(name, modifier)`; `*modifiers.Registry` implements it                                   |
| `NormalizePhone`  | The modifier: formats `string` and pointer-to-`string` values in E.164                            |
| `ErrInvalidPhone` | Cause of the `*modifiers.ModifierError` for numbers that do not parse or are invalid for the region |

## Parameters

| Parameter | Default | Description                                                         |
|-----------|---------|---------------------------------------------------------------------|
| `region`  | `RU`    | ISO 3166-1 region used for numbers without a country code; any case |

Spaces, `-`, `(`, `)`, `.` and `_` are removed before parsing; empty values are left as they are. Results, including failures, are kept in a
bounded FIFO cache of 4096 entries; inputs longer than 256 bytes with their region are not cached.
