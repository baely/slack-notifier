FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
RUN go build -o /app/main .

FROM alpine

WORKDIR /app

COPY --from=builder /app/main .

CMD ["/app/main"]
