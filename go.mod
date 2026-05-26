module mist

go 1.26.3

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/miekg/dns v1.1.72
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/crypto v0.49.0
	golang.zx2c4.com/wireguard v0.0.0-20260522210424-ecfc5a8d5446
	gvisor.dev/gvisor v0.0.0-20250503011706-39ed1f5ac29c
)

require (
	github.com/google/btree v1.1.2 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
)

require (
	MistCore v0.0.0
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.45.0
	golang.org/x/text v0.36.0 // indirect
)

replace MistCore => ./MistCore
