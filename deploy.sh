#!/bin/bash

set -e

echo "=========================================="
echo "Jarvis 智能发布系统 - 部署脚本"
echo "=========================================="

# Configuration
CONFIG_FILE="${CONFIG_FILE:-cmd/jarvis/etc/jarvis.yaml}"
DATA_DIR="${DATA_DIR:-./data}"
PORT="${PORT:-8080}"

# Parse command line arguments
ACTION="${1:-start}"

# Function to build the application
build() {
    echo "正在构建后端..."
    cd cmd/jarvis && go build -o jarvis jarvis.go && cd ../..
    echo "✅ 后端构建完成"
    
    echo ""
    echo "正在构建前端..."
    cd web && npm install && npm run build && cd ..
    echo "✅ 前端构建完成"
}

# Function to start the service
start() {
    echo ""
    echo "正在启动 Jarvis 服务..."
    echo "配置文件: $CONFIG_FILE"
    echo "数据目录: $DATA_DIR"
    echo "服务端口: $PORT"
    echo ""
    
    # Create data directory if it doesn't exist
    mkdir -p "$DATA_DIR"
    
    # Start the service
    cd cmd/jarvis
    ./jarvis -f etc/jarvis.yaml
}

# Function to stop the service
stop() {
    echo "正在停止 Jarvis 服务..."
    pkill -f "jarvis -f" || echo "服务未运行"
    echo "✅ 服务已停止"
}

# Function to show status
status() {
    if pgrep -f "jarvis -f" > /dev/null; then
        echo "✅ Jarvis 服务正在运行"
        echo ""
        echo "进程信息:"
        ps aux | grep "jarvis -f" | grep -v grep
    else
        echo "❌ Jarvis 服务未运行"
    fi
}

# Main logic
case "$ACTION" in
    build)
        build
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop
        sleep 2
        start
        ;;
    status)
        status
        ;;
    deploy)
        build
        stop
        sleep 2
        start
        ;;
    *)
        echo "用法: $0 {build|start|stop|restart|status|deploy}"
        echo ""
        echo "命令说明:"
        echo "  build   - 构建后端和前端"
        echo "  start   - 启动服务"
        echo "  stop    - 停止服务"
        echo "  restart - 重启服务"
        echo "  status  - 查看服务状态"
        echo "  deploy  - 完整部署(构建+重启)"
        echo ""
        echo "环境变量:"
        echo "  CONFIG_FILE - 配置文件路径 (默认: cmd/jarvis/etc/jarvis.yaml)"
        echo "  DATA_DIR    - 数据目录路径 (默认: ./data)"
        echo "  PORT        - 服务端口 (默认: 8080)"
        exit 1
        ;;
esac
