FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./app.go ./app.go
COPY ./internal ./internal
COPY ./pkg ./pkg

RUN go mod download
RUN go build -o /app/main .

FROM alpine

WORKDIR /app

COPY --from=builder /app/main .

CMD ["/app/main"]
