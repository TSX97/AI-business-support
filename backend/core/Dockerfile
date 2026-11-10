FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/core ./cmd/core

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/core /usr/local/bin/core

EXPOSE 8081

ENTRYPOINT ["core"]
