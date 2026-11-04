# go-normalize

Reflection-based normalization for Go structs using `normalize` tags. Built-in modifiers work without optional modules; phone normalization is opt-in through the `phone` module.

```go
type Contact struct { Name string `normalize:"trim,lowercase"` }
contact := Contact{Name: "  JANE "}
if err := normalize.Normalize(&contact); err != nil { /* handle error */ }
```

For phone numbers, create a `modifiers.Registry`, call `phone.Register(registry)`, then use `normalize.NormalizeWithRegistry`.
