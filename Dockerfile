# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# เพิ่ม dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source ทั้งหมด
COPY . .

# Compile binary
RUN go build -o main .

# 🏁 Final image (clean, no go toolchain)
FROM alpine:latest

WORKDIR /app

# เอาแค่ binary จาก build stage
COPY --from=builder /app/main .

EXPOSE 4001

CMD ["./main"]
