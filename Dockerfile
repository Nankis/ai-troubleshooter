FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

ARG SERVICE=investigation-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/${SERVICE}

FROM alpine:3.21

RUN adduser -D -H appuser
WORKDIR /app
COPY --from=builder /out/server /app/server
USER appuser

EXPOSE 8080
ENTRYPOINT ["/app/server"]
