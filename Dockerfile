FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

RUN swag init -g cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o mcp ./cmd/mcp
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o mlbackfill ./cmd/mlbackfill
RUN CGO_ENABLED=0 GOOS=linux go build -o sshserver ./cmd/ssh

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/mcp .
COPY --from=builder /app/migrate .
COPY --from=builder /app/mlbackfill .
COPY --from=builder /app/sshserver .

EXPOSE 8080
EXPOSE 2222

CMD ["./main"]
