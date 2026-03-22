module watchflare-agent

go 1.26.1

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/shirou/gopsutil/v4 v4.26.2
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.11
	howett.net/plist v1.0.1
	watchflare/shared v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
)

replace watchflare/shared => ../shared
