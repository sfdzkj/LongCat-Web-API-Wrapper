# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

RUN apk add --no-cache ca-certificates git

COPY go.mod ./
RUN go mod download

COPY . .
RUN go mod tidy

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/longcat-web-api .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata su-exec  && adduser -D -u 10001 app  && mkdir -p /app/data  && chown -R 10001:10001 /app
COPY --from=builder /out/longcat-web-api /app/longcat-web-api
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh /app/longcat-web-api
EXPOSE 8082
ENTRYPOINT ["/app/entrypoint.sh"]
