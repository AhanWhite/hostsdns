.PHONY: all build run gotool clean help

# 变量定义
BINARY="hostsdns"
OUTPUT="_output"

all: gotool build

build:
	mkdir -p ${OUTPUT}/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build -o ${OUTPUT}/bin/${BINARY}

run:
	@go run ./

gotool:
	go fmt ./
	go vet ./

clean:
	@rm -rf ${OUTPUT}

help:
	@echo "make - 格式化 Go 代码, 并编译生成二进制文件"
	@echo "make build - 编译 Go 代码, 生成二进制文件"
	@echo "make run - 直接运行 Go 代码"
	@echo "make clean - 移除二进制文件和 vim swap files"
	@echo "make gotool - 运行 Go 工具 'fmt' and 'vet'"