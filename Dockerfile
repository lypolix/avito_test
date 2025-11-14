# syntax=docker/dockerfile:1

FROM golang:1.25.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /server /app/server
COPY --from=builder /go/bin/migrate /app/migrate
COPY migrations /app/migrations
EXPOSE 8080
USER nonroot:nonroot

