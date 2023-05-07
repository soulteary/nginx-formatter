FROM golang:1.20.0-alpine3.16 as builder
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/community"    >> /etc/apk/repositories && \
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

FROM alpine:3.16
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/community"    >> /etc/apk/repositories && \
    echo "Asia/Shanghai" > /etc/timezone
RUN apk add openssl && rm -rf /var/cache/apk/*
WORKDIR /
COPY --from=Builder /build/nginx-formatter /bin/nginx-formatter
ENTRYPOINT ["nginx-formatter"]