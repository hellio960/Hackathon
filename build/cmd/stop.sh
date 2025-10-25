#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

BACKEND_PID_FILE="backend.pid"
FRONTEND_PID_FILE="frontend.pid"

echo "========================================"
echo "Jarvis 智能发布系统 - 停止脚本"
echo "========================================"

# 停止后端服务
stop_backend() {
    if [ -f "${BACKEND_PID_FILE}" ]; then
        PID=$(cat ${BACKEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo "停止后端服务 (PID: $PID)..."
            kill $PID
            
            # 等待进程结束
            for i in {1..10}; do
                if ! ps -p $PID > /dev/null 2>&1; then
                    echo -e "${GREEN}✓ 后端服务已停止${NC}"
                    rm -f ${BACKEND_PID_FILE}
                    return
                fi
                sleep 1
            done
            
            # 强制停止
            echo -e "${YELLOW}强制停止后端服务...${NC}"
            kill -9 $PID 2>/dev/null
            rm -f ${BACKEND_PID_FILE}
            echo -e "${GREEN}✓ 后端服务已强制停止${NC}"
        else
            echo -e "${YELLOW}后端服务未运行${NC}"
            rm -f ${BACKEND_PID_FILE}
        fi
    else
        echo -e "${YELLOW}未找到后端服务PID文件${NC}"
    fi
}

# 停止前端代理
stop_frontend() {
    if [ -f "${FRONTEND_PID_FILE}" ]; then
        PID=$(cat ${FRONTEND_PID_FILE})
        if ps -p $PID > /dev/null 2>&1; then
            echo "停止前端代理 (PID: $PID)..."
            kill $PID
            
            # 等待进程结束
            for i in {1..10}; do
                if ! ps -p $PID > /dev/null 2>&1; then
                    echo -e "${GREEN}✓ 前端代理已停止${NC}"
                    rm -f ${FRONTEND_PID_FILE}
                    return
                fi
                sleep 1
            done
            
            # 强制停止
            echo -e "${YELLOW}强制停止前端代理...${NC}"
            kill -9 $PID 2>/dev/null
            rm -f ${FRONTEND_PID_FILE}
            echo -e "${GREEN}✓ 前端代理已强制停止${NC}"
        else
            echo -e "${YELLOW}前端代理未运行${NC}"
            rm -f ${FRONTEND_PID_FILE}
        fi
    else
        echo -e "${YELLOW}未找到前端代理PID文件${NC}"
    fi
}

# 主函数
main() {
    stop_backend
    stop_frontend
    
    echo ""
    echo "========================================"
    echo -e "${GREEN}所有服务已停止${NC}"
    echo "========================================"
}

main
