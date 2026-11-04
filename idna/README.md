# idna

```go
import "github.com/altessa-s/go-normalize/idna"
```

Opt-in modifier that converts internationalized domain names to the ASCII form used by DNS and mail servers, for [`normalize`](../README.md)
tags. It is a separate module so that `golang.org/x/net` stays out of the root module's requirements.

## Usage

```go
registry := modifiers.NewRegistry()
idna.Register(registry)

type Contact struct {
	Email   string `normalize:"trim,email,idna"`
	Website string `normalize:"trim,url,idna"`
}
err := normalize.NormalizeWithRegistry(&contact, registry)
```

`idna` applies the UTS #46 lookup rules: the domain is lower-cased and non-ASCII labels become punycode, so `user@Bücher.de` becomes
`user@xn--bcher-kva.de`. It converts a bare domain, the domain after the last `@` of an email address, or the host of an absolute URL; IP
literals, ports, paths, queries and fragments are kept. Combine it with the built-in `email` or `url` for the rest of the address.

## Key types

| Identifier         | Description                                                                                           |
|--------------------|-------------------------------------------------------------------------------------------------------|
| `Register`         | Adds the `idna` modifier to a `Registrar`                                                             |
| `Registrar`        | `Register(name, modifier)`; `*modifiers.Registry` implements it                                       |
| `ToASCII`          | The modifier; applies to `string` values and pointers to them                                         |
| `ErrInvalidDomain` | Cause of the `*modifiers.ModifierError` for a domain that is not a valid IDN; the value is not echoed |
