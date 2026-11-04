module github.com/altessa-s/go-normalize/idna

go 1.27.0

toolchain go1.27.3

require (
	github.com/altessa-s/go-normalize v0.1.0
	golang.org/x/net v0.62.0
)

require golang.org/x/text v0.43.0 // indirect

replace github.com/altessa-s/go-normalize => ../
