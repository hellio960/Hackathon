.PHONY: all build clean test run install deps docker-build docker-run help

APP_NAME=deployer
VERSION?=latest
DOCKER_IMAGE=intelligent-deployment-system
DOCKER_TAG=$(VERSION)

all: deps build

help:
	@echo "智能发布系统 - Makefile 使用说明"
	@echo ""
	@echo "可用命令:"
	@echo "  make deps          - 安装 Go 依赖"
	@echo "  make build         - 编译主程序"
	@echo "  make build-all     - 编译所有示例程序"
	@echo "  make test          - 运行测试"
	@echo "  make run           - 运行主程序"
	@echo "  make clean         - 清理编译产物"
	@echo "  make install       - 安装到系统路径"
	@echo "  make docker-build  - 构建 Docker 镜像"
	@echo "  make docker-run    - 运行 Docker 容器"
	@echo "  make k8s-deploy    - 部署到 Kubernetes"
	@echo "  make k8s-delete    - 从 Kubernetes 删除"
	@echo ""

deps:
	@echo "==> 安装依赖..."
	go mod download
	go mod tidy

build:
	@echo "==> 编译主程序..."
	go build -o bin/$(APP_NAME) cmd/deployer/main.go
	@echo "编译完成: bin/$(APP_NAME)"

build-all: build
	@echo "==> 编译所有示例程序..."
	go build -o bin/k8s_deployment examples/k8s_deployment.go
	go build -o bin/physical_deployment examples/physical_deployment.go
	go build -o bin/parallel_deployment examples/parallel_deployment.go
	@echo "所有程序编译完成"

test:
	@echo "==> 运行测试..."
	go test -v ./...

run: build
	@echo "==> 运行主程序..."
	./bin/$(APP_NAME)

run-k8s: build-all
	@echo "==> 运行 K8S 部署示例..."
	./bin/k8s_deployment

run-physical: build-all
	@echo "==> 运行物理机部署示例..."
	./bin/physical_deployment

run-parallel: build-all
	@echo "==> 运行并行部署示例..."
	./bin/parallel_deployment

clean:
	@echo "==> 清理编译产物..."
	rm -rf bin/
	rm -f $(APP_NAME)
	@echo "清理完成"

install: build
	@echo "==> 安装到系统路径..."
	cp bin/$(APP_NAME) /usr/local/bin/
	@echo "安装完成: /usr/local/bin/$(APP_NAME)"

docker-build:
	@echo "==> 构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker 镜像构建完成: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run:
	@echo "==> 运行 Docker 容器..."
	docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG)

k8s-deploy:
	@echo "==> 部署到 Kubernetes..."
	kubectl apply -f deployments/kubernetes/
	@echo "部署完成"

k8s-delete:
	@echo "==> 从 Kubernetes 删除..."
	kubectl delete -f deployments/kubernetes/
	@echo "删除完成"

k8s-status:
	@echo "==> 查看部署状态..."
	kubectl get pods -l app=intelligent-deployment-system
	kubectl get svc -l app=intelligent-deployment-system

fmt:
	@echo "==> 格式化代码..."
	go fmt ./...

lint:
	@echo "==> 运行代码检查..."
	golint ./...

vet:
	@echo "==> 运行 go vet..."
	go vet ./...
