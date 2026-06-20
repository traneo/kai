module kaiplatform.com/agent

go 1.25.0

replace kaiplatform.com/gen => ../gen

replace kaiplatform.com/observability-sdk => ../../kai-observability-sdk/go

require (
	google.golang.org/grpc v1.81.1
	kaiplatform.com/gen v0.0.0-00010101000000-000000000000
	kaiplatform.com/observability-sdk v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
