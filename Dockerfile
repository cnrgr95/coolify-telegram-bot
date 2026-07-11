FROM golang:1.23-bookworm

WORKDIR /app

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    build-essential curl ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN GOFLAGS="-mod=mod" go mod download

COPY . .

RUN GOTOOLCHAIN=auto go generate ./...

RUN CGO_ENABLED=0 go build -o bot .

EXPOSE 8080

CMD ["./bot"]
