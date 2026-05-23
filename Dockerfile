FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . /app
EXPOSE 1448
RUN go build -o bot main.go
CMD ["./bot"]