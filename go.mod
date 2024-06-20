module kent-control-interface

go 1.14

require (
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/kr/text v0.2.0 // indirect
	gitlab.com/karakuritech/dk/kent v0.3.0
	gitlab.com/karakuritech/dk/utensils v0.8.1
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
	google.golang.org/protobuf v1.28.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace (
	gitlab.com/karakuritech/dk/kent => ../kent
	gitlab.com/karakuritech/dk/utensils => gitlab.com/karakuritech/dk/utensils.git v0.9.3
)
