#!/bin/bash

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

BACKEND_DIR="backend"
FRONTEND_DIR="frontend"
BACKEND_PID_FILE="backend.pid"
FRONTEND_PID_FILE="frontend.pid"

echo "========================================"
echo "Jarvis 智能发布系统 - 启动脚本"
echo "========================================"

# 检查配置文件
check_config() {
    if [ ! -f "${BACKEND_DIR}/jarvis.yaml" ]; then
        if [ -f "${BACKEND_DIR}/jarvis.yaml.example" ]; then
            echo -e "${YELLOW}警告: 未找到配置文件 jarvis.yaml${NC}"
            echo -e "${YELLOW}请复制 jarvis.yaml.example 并修改为 jarvis.yaml${NC}"
            echo ""
            read -p "是否使用示例配置启动？(y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                cp ${BACKEND_DIR}/jarvis.yaml.example ${BACKEND_DIR}/jarvis.yaml
                echo -e "${GREEN}已复制示例配置文件${NC}"
            else
                echo -e "${RED}启动取消${NC}"
                exit 1
            fi
        else
            echo -e "${RED}错误: 未找到配置文件${NC}"
            exit 1
        fi
    fi
}

# 启动后端服务
start_backend() {
    echo ""
    echo "启动后端服务..."
    
    if [ -f "${BACKEND_PID_FILE}" ]; then
        PID=$(cat ${BACKEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}后端服务已在运行 (PID: $PID)${NC}"
            return
        fi
    fi
    
    cd ${BACKEND_DIR}
    nohup ./jarvis -f jarvis.yaml > jarvis.log 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > ../${BACKEND_PID_FILE}
    cd ..
    
    sleep 2
    
    if ps -p $BACKEND_PID > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 后端服务启动成功 (PID: $BACKEND_PID)${NC}"
        echo "日志文件: ${BACKEND_DIR}/jarvis.log"
    else
        echo -e "${RED}✗ 后端服务启动失败${NC}"
        echo "请查看日志: ${BACKEND_DIR}/jarvis.log"
        exit 1
    fi
}

# 启动前端代理（可选）
start_frontend() {
    if [ ! -f "${FRONTEND_DIR}/proxy.js" ]; then
        echo -e "${YELLOW}未找到前端代理文件，跳过前端启动${NC}"
        return
    fi
    
    if ! command -v node &> /dev/null; then
        echo -e "${YELLOW}未找到Node.js环境，跳过前端代理启动${NC}"
        echo "您可以直接通过Nginx等Web服务器部署前端静态文件"
        return
    fi
    
    echo ""
    echo "启动前端代理服务..."
    
    if [ -f "${FRONTEND_PID_FILE}" ]; then
        PID=$(cat ${FRONTEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "${YELLOW}前端代理已在运行 (PID: $PID)${NC}"
            return
        fi
    fi
    
    cd ${FRONTEND_DIR}
    
    # 检查依赖
    if [ ! -d "node_modules" ]; then
        echo "安装Node.js依赖..."
        npm install
    fi
    
    nohup node proxy.js > proxy.log 2>&1 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > ../${FRONTEND_PID_FILE}
    cd ..
    
    sleep 2
    
    if ps -p $FRONTEND_PID > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 前端代理启动成功 (PID: $FRONTEND_PID)${NC}"
        echo "日志文件: ${FRONTEND_DIR}/proxy.log"
        echo "访问地址: http://localhost:3000"
    else
        echo -e "${RED}✗ 前端代理启动失败${NC}"
        echo "请查看日志: ${FRONTEND_DIR}/proxy.log"
    fi
}

# 显示状态
show_status() {
    echo ""
    echo "========================================"
    echo "服务状态"
    echo "========================================"
    
    if [ -f "${BACKEND_PID_FILE}" ]; then
        PID=$(cat ${BACKEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "后端服务: ${GREEN}运行中${NC} (PID: $PID)"
        else
            echo -e "后端服务: ${RED}已停止${NC}"
        fi
    else
        echo -e "后端服务: ${RED}未启动${NC}"
    fi
    
    if [ -f "${FRONTEND_PID_FILE}" ]; then
        PID=$(cat ${FRONTEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo -e "前端代理: ${GREEN}运行中${NC} (PID: $PID)"
        else
            echo -e "前端代理: ${RED}已停止${NC}"
        fi
    else
        echo -e "前端代理: ${RED}未启动${NC}"
    fi
    
    echo "========================================"
}

# 主函数
main() {
    check_config
    start_backend
    start_frontend
    show_status
    
    echo ""
    echo "使用 './stop.sh' 停止服务"
}

main
