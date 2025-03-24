
# Build stage
FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go build -o badgeradmin .

# Runtime stage
FROM debian:bullseye-slim

WORKDIR /app
COPY --from=builder /app/badgeradmin .
EXPOSE 8080

ENV BADGER_ADMIN_USER=admin
ENV BADGER_ADMIN_PASS=botfluence

CMD ["./badgeradmin"]
