module kent-control-interface

go 1.14

require (
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/iwdfryer/kent v0.0.4
	github.com/iwdfryer/utensils v0.0.2
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.28.0
)

replace (
	github.com/iwdfryer/kent => ../gateway/kent
	github.com/iwdfryer/utensils => ../gateway/utensils
)
