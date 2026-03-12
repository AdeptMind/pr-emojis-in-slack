FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pr-emojis-in-slack ./cmd/pr-emojis-in-slack

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /pr-emojis-in-slack /pr-emojis-in-slack

ENTRYPOINT ["/pr-emojis-in-slack"]
