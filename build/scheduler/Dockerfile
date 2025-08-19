FROM golang:latest AS builder
LABEL authors="trint"
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/scheduler ./cmd/scheduler/

FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    logrotate \
    cron \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY ./build/scheduler/scheduler-logrotate /etc/logrotate.d/

RUN chmod 0644 /etc/logrotate.d/

WORKDIR /app
COPY --from=builder /bin/scheduler ./bin/scheduler
RUN mkdir -p ./log
CMD ["./bin/scheduler"]