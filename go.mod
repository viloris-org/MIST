module mist

go 1.26.3

require (
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/crypto v0.40.0
)

require (
	MistCore v0.0.0
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)

replace MistCore => ./MistCore
