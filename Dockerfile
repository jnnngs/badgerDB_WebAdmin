# Build stage with static binary
FROM golang:1.23 AS builder

WORKDIR /app
COPY . .

# Build statically
ENV CGO_ENABLED=0
RUN go build -o badgeradmin .

# Runtime stage
FROM scratch

WORKDIR /app
COPY --from=builder /app/badgeradmin .

EXPOSE 8080

ENV BADGER_ADMIN_USER=admin
ENV BADGER_ADMIN_PASS=botfluence

CMD ["./badgeradmin"]
