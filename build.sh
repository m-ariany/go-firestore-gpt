# Needs to be replaced with a build pipeline

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -o buywise-go.exe main.go

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o linux-amd64-buywise-go main.go

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -o darwin-amd64-buywise-go main.go

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -o darwin-arm64-buywise-go main.go