FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /simulator ./cmd/simulator

FROM alpine:3.21
COPY --from=builder /simulator /simulator
EXPOSE 19100
CMD ["/simulator"]
