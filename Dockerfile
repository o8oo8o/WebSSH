# 第一阶段：构建
FROM golang:1.23.4 AS builder
WORKDIR /mnt
COPY gossh ./gossh
RUN cd /mnt/gossh && CGO_ENABLED=0 go build -o /mnt/gowebssh

# 第二阶段：运行
FROM alpine:3.21.0
WORKDIR /root/
VOLUME /var/lib/webssh
EXPOSE 8899
COPY --from=builder /mnt/gowebssh /bin/gowebssh
CMD ["/bin/gowebssh", "-WorkDir", "/var/lib/webssh"]