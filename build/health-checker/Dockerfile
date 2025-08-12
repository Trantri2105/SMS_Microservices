FROM golang:latest AS builder
LABEL authors="trint"
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/checker ./cmd/health-checker/

FROM ubuntu:latest

RUN apt-get update && apt-get install -y dos2unix \
    logrotate \
    cron \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY ./build/health-checker/health-checker-logrotate /etc/logrotate.d/

RUN dos2unix /etc/logrotate.d/health-checker-logrotate
RUN chmod 0644 /etc/logrotate.d/

WORKDIR /app
COPY --from=builder /bin/checker ./bin/checker
RUN mkdir -p ./log
CMD ["./bin/checker"]