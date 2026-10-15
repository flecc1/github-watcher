FROM golang:1.27.2-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /github-watcher .

FROM alpine:3.24
RUN apk --no-cache add ca-certificates
COPY --from=builder /github-watcher /usr/local/bin/github-watcher
CMD ["github-watcher"]