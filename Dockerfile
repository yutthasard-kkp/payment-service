FROM golang:1.24.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main .

FROM alpine:3.21.3 AS runner

WORKDIR /app

RUN addgroup -g 1001 nonroot && \
    adduser -D -u 1001 -G nonroot nonroot

COPY --from=builder --chmod=0500 /app/main .

RUN chown nonroot:nonroot /app/main

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/main"]
