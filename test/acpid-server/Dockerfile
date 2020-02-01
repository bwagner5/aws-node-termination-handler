# Build the manager binary
FROM golang:1-alpine as builder

## GOLANG env
ARG GOPROXY="https://proxy.golang.org,direct"
ARG GO111MODULE="on"
ARG CGO_ENABLED=0
ARG GOOS=linux 
ARG GOARCH=amd64 

# Copy go.mod and download dependencies
WORKDIR /acpid-server

# Build
COPY . .
RUN go build -a -o acpid-server cmd/acpid-server.go
# In case the target is build for testing:
# $ docker build  --target=builder -t test .
ENTRYPOINT ["acpid-server"]

# Copy the controller-manager into a thin image
FROM amazonlinux:2 as amazonlinux
FROM scratch
WORKDIR /
COPY --from=builder /acpid-server/acpid-server .
COPY --from=amazonlinux /etc/ssl/certs/ca-bundle.crt /etc/ssl/certs/
COPY THIRD_PARTY_LICENSES .
ENTRYPOINT ["/acpid-server"]
