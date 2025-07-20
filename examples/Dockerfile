FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY . .


RUN cd examples && go mod download


RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./examples/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]

