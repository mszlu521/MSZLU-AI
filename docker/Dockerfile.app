FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 设置Go模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 复制go.work（利用缓存层）
COPY go.work go.work.sum ./

# 复制各模块的go.mod和go.sum（利用缓存层）
# 使用通配符，不存在的文件会被忽略（目标目录以/结尾）
COPY app/go.* ./app/
COPY common/go.* ./common/
COPY core/go.* ./core/
COPY model/go.* ./model/
COPY a2a-server/go.* ./a2a-server/
COPY mcp-server/go.* ./mcp-server/

# 下载所有依赖
RUN go work sync

# 复制所有源代码
COPY app ./app
COPY common ./common
COPY core ./core
COPY model ./model
COPY a2a-server ./a2a-server
COPY mcp-server ./mcp-server
# 编译应用（在app目录下执行）
WORKDIR /build/app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/main main.go

# 使用最小的基础镜像
FROM alpine:latest

# 安装ca-certificates和wget以支持HTTPS请求和健康检查
RUN apk --no-cache add ca-certificates wget tzdata
# 设置时区环境变量（确保Go程序能正确识别时区）
ENV TZ=Asia/Shanghai
# 创建非root用户
RUN adduser -D -s /bin/sh appuser

WORKDIR /app

# 从builder阶段复制二进制文件
COPY --from=builder /build/main .

# 创建配置目录（如果不存在）
RUN mkdir -p /app/etc

# 复制默认配置文件（作为fallback）
COPY ./app/etc ./etc

# 更改文件所有权
RUN chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8888

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8888/health || exit 1

# 启动应用
# 默认使用 etc/config.yml，可以通过 -c 参数指定其他配置文件
# 例如: ./main -c /app/etc/config.yml
ENTRYPOINT ["./main"]
CMD ["-c", "/app/etc/config.yml"]
