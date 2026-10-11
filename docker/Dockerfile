FROM golang:1.27.2-alpine3.24 AS builder
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

FROM alpine:3.24
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.24/community"    >> /etc/apk/repositories && \
    echo "Asia/Shanghai" > /etc/timezone
WORKDIR /
COPY --from=builder /build/nginx-formatter /bin/nginx-formatter
ENTRYPOINT ["nginx-formatter"]