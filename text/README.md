# text

```go
import "github.com/altessa-s/go-normalize/text"
```

Opt-in Unicode modifiers for [`normalize`](../README.md) tags, in their own module so that `golang.org/x/text` stays out of the root module's
requirements.

## Usage

```go
registry := modifiers.NewRegistry()
text.Register(registry)

type Account struct {
	Login string `normalize:"trim,nfkc,fold"`
	Name  string `normalize:"trim,nfc,title(lang=en)"`
}
err := normalize.NormalizeWithRegistry(&account, registry)
```

## Modifiers

| Name               | Effect                                                                                                   |
|--------------------|----------------------------------------------------------------------------------------------------------|
| `nfc`              | Canonical composition: a letter plus a combining mark becomes the precomposed letter; safe on any text   |
| `nfkc`             | Compatibility composition: also maps full-width letters, ligatures and superscripts to plain forms       |
| `fold`             | Unicode case folding for caseless keys (`Straße` and `STRASSE` become `strasse`); not for display        |
| `title(lang=xx)`   | Title case by the rules of a BCP 47 tag (`nl`: `IJsland`, `tr`: `İstanbul`); language neutral without it |

All four apply to `string` values and pointers to them, named types included. An invalid `lang` fails with `modifiers.ErrInvalidParameter`.
Use `nfc` on text that is compared, indexed or kept unique; use `nfkc` and `fold` for identifiers and search keys, since they change how text
looks.

## Key types

| Identifier  | Description                                                     |
|-------------|-----------------------------------------------------------------|
| `Register`  | Adds `nfc`, `nfkc`, `fold` and `title` to a `Registrar`         |
| `Registrar` | `Register(name, modifier)`; `*modifiers.Registry` implements it |
