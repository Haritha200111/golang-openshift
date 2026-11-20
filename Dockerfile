# Build a static Linux binary. Pin the Go version to match go.mod.
FROM golang:1.23.4 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/task-api .

# UBI minimal provides a small Red Hat-based runtime with CA certificates.
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5
WORKDIR /
COPY --from=build /out/task-api /usr/local/bin/task-api
USER 1001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/task-api"]
