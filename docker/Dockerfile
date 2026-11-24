FROM golang:1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/community"    >> /etc/apk/repositories && \
    echo "Asia/Shanghai" > /etc/timezone
RUN apk add upx
WORKDIR /build
ENV CGO_ENABLED=0
COPY ./go.mod .
COPY ./go.sum .
RUN go mod tidy && go mod download
COPY ./ .
RUN go build -ldflags "-w -s"
RUN upx -9 -o nginx-formatter.minify nginx-formatter && mv nginx-formatter.minify nginx-formatter

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/community"    >> /etc/apk/repositories && \
    echo "Asia/Shanghai" > /etc/timezone
WORKDIR /
COPY --from=builder /build/nginx-formatter /bin/nginx-formatter
ENTRYPOINT ["nginx-formatter"]