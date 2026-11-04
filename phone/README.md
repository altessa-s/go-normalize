# phone

```go
import "github.com/altessa-s/go-normalize/phone"
```

Opt-in phone-number modifier for [`normalize`](../README.md) tags, in its own module so that `github.com/nyaruka/phonenumbers` stays out of
the root module's requirements.

## Usage

```go
registry := modifiers.DefaultRegistry()
if err := phone.Register(registry, "US"); err != nil {
	return err
}

type Contact struct {
	Mobile string `normalize:"trim,phone"`
	Office string `normalize:"trim,phone(region=DE)"`
}
err := normalize.NormalizeWithRegistry(&contact, registry)
```

## Key types

| Identifier             | Description                                                                                 |
|------------------------|---------------------------------------------------------------------------------------------|
| `Register`             | Adds the `phone` modifier with a required default region to a `modifiers.Registrar`         |
| `New`                  | Returns the modifier for a default region, for registering under another name               |
| `ErrInvalidPhone`      | Cause of the `*modifiers.Error` for numbers that do not parse or are invalid for the region |
| `ErrUnsupportedRegion` | A default or tag region without phone number metadata                                       |

## Parameters

| Parameter | Default                         | Description                                                         |
|-----------|---------------------------------|---------------------------------------------------------------------|
| `region`  | The region passed to `Register` | ISO 3166-1 region used for numbers without a country code; any case |

The modifier formats `string` values and pointer chains ending in one, named types included, in E.164. Spaces, `-`, `(`, `)`, `.` and `_`
are removed before parsing; empty values are left as they are. Results, including failures, are kept in a bounded FIFO cache of 4096
entries; inputs longer than 256 bytes with their region are not cached.

## License

The modifier relies on [`github.com/nyaruka/phonenumbers`](https://github.com/nyaruka/phonenumbers), MIT licensed, which embeds phone
number metadata generated from Google's [libphonenumber](https://github.com/google/libphonenumber) under the Apache License 2.0. Binaries
that include this module distribute that metadata, so carry its notice along with the MIT notice of go-normalize.
