# normalizetest

```go
import "github.com/altessa-s/go-normalize/normalizetest"
```

Test helpers for consumers of go-normalize.

## Key types

| Identifier          | Description                                                  |
|---------------------|--------------------------------------------------------------|
| `RequireNormalized` | Calls a normalize function and fails the test with its error |

## Usage

```go
normalizetest.RequireNormalized(t, normalize.Normalize, &contact)
```
