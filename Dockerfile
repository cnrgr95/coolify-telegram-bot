FROM golang:1.26

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential curl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate

RUN CGO_ENABLED=1 go build -o bot .

CMD ["./bot"]

EXPOSE 8080
