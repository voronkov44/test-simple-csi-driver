FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN go build -o driver main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/driver .

RUN chmod +x /app/driver
RUN mkdir -p /csi

CMD ["/app/driver"]
