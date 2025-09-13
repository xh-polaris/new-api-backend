FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)'" -o one-api

FROM alpine

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata ffmpeg \
    && update-ca-certificates


# 设置默认环境变量
ENV SQL_DSN="root:111@tcp(mysql.mysql:3306)/new-api?charset=utf8&parseTime=True&loc=UTC" \
    REDIS_CONN_STRING="redis://redis-master.redis:6379" \
    TZ="Asia/Shanghai" \
    ERROR_LOG_ENABLED="true" \
    SESSION_SECRET="xh-polaris"



COPY --from=builder /build/one-api /
EXPOSE 8080
WORKDIR /data
ENTRYPOINT ["/one-api"]
