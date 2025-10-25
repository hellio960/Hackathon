#!/bin/bash

set -e

echo "========================================"
echo "智能发布系统构建脚本"
echo "========================================"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 支持的平台列表 (格式: OS:ARCH)
PLATFORMS=("linux:amd64" "darwin:amd64" "darwin:arm64")

# 清理构建产物
cleanup() {
    local package_dir=$1

    if [ -z "$package_dir" ]; then
        echo -e "${RED}错误: 请提供部署包目录作为参数${NC}"
        exit 1
    fi

    echo ""
    echo "========================================"
    echo "清理构建产物..."
    echo "========================================"

    rm -rf "$package_dir"
    echo "已删除: $package_dir"

    # 删除各平台的后端二进制文件
    for platform in "${PLATFORMS[@]}"; do
        IFS=':' read -r os arch <<< "$platform"
        binary_name="jarvis-${os}-${arch}"
        if [ -f "cmd/jarvis/${binary_name}" ]; then
            rm -f "cmd/jarvis/${binary_name}"
            echo "已删除: cmd/jarvis/${binary_name}"
        fi
    done

    echo -e "${GREEN}✓ 清理完成${NC}"
}

# 检查Go环境
check_go() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未找到Go环境，请先安装Go 1.21或更高版本${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Go环境检查通过: $(go version)${NC}"
}

# 检查Node.js环境（可选，用于前端代理）
check_node() {
    if ! command -v node &> /dev/null; then
        echo -e "${YELLOW}警告: 未找到Node.js环境，前端代理功能将不可用${NC}"
        echo -e "${YELLOW}如需使用前端代理，请安装Node.js 14或更高版本${NC}"
        return 1
    fi
    echo -e "${GREEN}✓ Node.js环境检查通过: $(node -v)${NC}"
    return 0
}

# 构建后端 (参数: OS ARCH)
build_backend() {
    local os=$1
    local arch=$2
    
    echo ""
    echo "========================================"
    echo "开始构建后端服务 (${os}/${arch})..."
    echo "========================================"
    
    cd cmd/jarvis
    
    # 构建
    echo "编译后端服务 (${os}/${arch})..."
    binary_name="jarvis-${os}-${arch}"
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -o ${binary_name} jarvis.go
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 后端构建成功 (${os}/${arch})${NC}"
        echo "二进制文件: cmd/jarvis/${binary_name}"
    else
        echo -e "${RED}✗ 后端构建失败 (${os}/${arch})${NC}"
        exit 1
    fi
    
    cd ../..
}

# 准备前端文件
prepare_frontend() {
    echo ""
    echo "========================================"
    echo "准备前端文件..."
    echo "========================================"
    
    if [ ! -d "web" ]; then
        echo -e "${RED}错误: 未找到web目录${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ 前端文件检查通过${NC}"
    echo "前端文件位置: web/"
}

# 安装Node.js依赖（可选）
install_node_deps() {
    if check_node; then
        echo ""
        echo "========================================"
        echo "安装Node.js依赖（用于前端代理）..."
        echo "========================================"
        
        cd web
        if [ ! -f "package.json" ]; then
            echo "创建package.json..."
            cat > package.json <<EOF
{
  "name": "jarvis-web",
  "version": "1.0.0",
  "description": "Jarvis智能发布系统前端",
  "main": "proxy.js",
  "scripts": {
    "start": "node proxy.js",
    "start-simple": "node simple-proxy.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "http-proxy-middleware": "^2.0.6"
  }
}
EOF
        fi
        
        if command -v npm &> /dev/null; then
            npm install
            echo -e "${GREEN}✓ Node.js依赖安装成功${NC}"
        else
            echo -e "${YELLOW}警告: npm未找到，跳过依赖安装${NC}"
        fi
        
        cd ..
    fi
}

# 创建部署包 (参数: OS ARCH TIMESTAMP)
create_deploy_package() {
    local os=$1
    local arch=$2
    local timestamp=$3
    
    echo ""
    echo "========================================"
    echo "创建部署包 (${os}/${arch})..."
    echo "========================================"
    
    DEPLOY_DIR="deploy"
    binary_name="jarvis-${os}-${arch}"
    platform_name="${os}-${arch}"
    PACKAGE_NAME="jarvis-release-${platform_name}-${timestamp}"
    
    # 清理旧的部署目录
    rm -rf ${DEPLOY_DIR}/${PACKAGE_NAME}
    mkdir -p ${DEPLOY_DIR}/${PACKAGE_NAME}
    
    # 复制后端文件
    echo "复制后端文件..."
    mkdir -p ${DEPLOY_DIR}/${PACKAGE_NAME}/backend/etc
    cp cmd/jarvis/${binary_name} ${DEPLOY_DIR}/${PACKAGE_NAME}/backend/jarvis
    cp cmd/jarvis/etc/jarvis-api.yaml ${DEPLOY_DIR}/${PACKAGE_NAME}/backend/etc/jarvis-api.yaml
    cp cmd/jarvis/etc/jarvis.yaml ${DEPLOY_DIR}/${PACKAGE_NAME}/backend/etc/jarvis.yaml
    
    # 复制前端文件
    echo "复制前端文件..."
    mkdir -p ${DEPLOY_DIR}/${PACKAGE_NAME}/frontend
    cp -r web/* ${DEPLOY_DIR}/${PACKAGE_NAME}/frontend/
    
    # 复制文档
    echo "复制文档..."
    mkdir -p ${DEPLOY_DIR}/${PACKAGE_NAME}/doc
    cp -r doc/DEPLOY.md ${DEPLOY_DIR}/${PACKAGE_NAME}/doc/
    
    # 复制启动脚本
    if [ -f "start.sh" ]; then
        cp start.sh ${DEPLOY_DIR}/${PACKAGE_NAME}/
        chmod +x ${DEPLOY_DIR}/${PACKAGE_NAME}/start.sh
    fi
    
    if [ -f "stop.sh" ]; then
        cp stop.sh ${DEPLOY_DIR}/${PACKAGE_NAME}/
        chmod +x ${DEPLOY_DIR}/${PACKAGE_NAME}/stop.sh
    fi
    
    # 创建README
    cat > ${DEPLOY_DIR}/${PACKAGE_NAME}/README.md <<EOF
# Jarvis 智能发布系统部署包 (${platform_name})

构建时间: ${timestamp}

## 目录结构

\`\`\`
.
├── backend/              # 后端服务
│   ├── jarvis           # 后端可执行文件
│   └── jarvis.yaml.example  # 配置文件示例
├── frontend/            # 前端静态文件
│   ├── *.html          # HTML页面
│   ├── css/            # 样式文件
│   ├── js/             # JavaScript文件
│   ├── proxy.js        # 代理服务器
│   └── package.json    # Node.js依赖配置
├── doc/                # 文档
│   └── DEPLOY.md       # 部署文档
├── start.sh            # 启动脚本
├── stop.sh             # 停止脚本
└── README.md           # 本文件
\`\`\`

## 快速开始

详细部署说明请参考: doc/DEPLOY.md
EOF
    
    # 打包
    echo "打包部署文件..."
    cd ${DEPLOY_DIR}
    tar -czf ${PACKAGE_NAME}.tar.gz ${PACKAGE_NAME}
    
    echo -e "${GREEN}✓ 部署包创建成功 (${os}/${arch})${NC}"
    echo "部署包位置: ${DEPLOY_DIR}/${PACKAGE_NAME}.tar.gz"
    echo "解压后目录: ${DEPLOY_DIR}/${PACKAGE_NAME}/"
    
    cd ..
}

# 主函数
main() {
    # 检查环境
    check_go
    prepare_frontend
    install_node_deps
    
    # 获取时间戳
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    echo ""
    echo "========================================"
    echo "开始构建所有平台..."
    echo "========================================"

    # 下载依赖
    echo "下载Go依赖..."
    go mod download

    # 为每个平台构建和打包
    for platform in "${PLATFORMS[@]}"; do
        IFS=':' read -r os arch <<< "$platform"
        
        # 构建后端
        build_backend ${os} ${arch}
        
        # 创建部署包
        create_deploy_package ${os} ${arch} ${TIMESTAMP}
    done

    echo ""
    echo "========================================"
    echo "所有平台构建完成!"
    echo "========================================"
    
    # 显示生成的包
    echo "生成的部署包:"
    for platform in "${PLATFORMS[@]}"; do
        IFS=':' read -r os arch <<< "$platform"
        platform_name="${os}-${arch}"
        PACKAGE_NAME="jarvis-release-${platform_name}-${TIMESTAMP}"
        echo "  - deploy/${PACKAGE_NAME}.tar.gz"

        # 清理构建产物
        cleanup "deploy/${PACKAGE_NAME}"
    done
}

# 执行主函数
main "$@"
