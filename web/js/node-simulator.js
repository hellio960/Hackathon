class NodeSimulator {
    constructor() {
        this.nodes = new Map(); // nodeId -> node data
        this.isRunning = false;
        this.intervalIds = new Map(); // nodeId -> intervalId
        this.init();
    }

    init() {
        this.bindEvents();
    }

    bindEvents() {
        // 开始模拟
        document.getElementById('startSimulationBtn').addEventListener('click', () => {
            this.startSimulation();
        });

        // 随机选择节点
        document.getElementById('randomSelectBtn').addEventListener('click', () => {
            this.randomSelectNodes();
        });

        // 停止模拟
        document.getElementById('stopSimulationBtn').addEventListener('click', () => {
            this.stopSimulation();
        });
    }

    // 添加节点日志
    addNodeLog(nodeId, message, type = 'info') {
        const logsContainer = document.getElementById(`node-logs-${nodeId}`);
        if (!logsContainer) return;

        const timestamp = new Date().toLocaleTimeString('zh-CN', { hour12: false });
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry log-${type}`;
        logEntry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span>${message}`;
        logsContainer.appendChild(logEntry);
        logsContainer.scrollTop = logsContainer.scrollHeight;
    }

    // 清空节点日志
    clearNodeLogs(nodeId) {
        const logsContainer = document.getElementById(`node-logs-${nodeId}`);
        if (logsContainer) {
            logsContainer.innerHTML = '';
        }
    }

    // 随机选择节点
    async randomSelectNodes() {
        try {
            const response = await fetch('/api/v1/nodes/search', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    size: 10,
                    status: 'online'
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();

            if (!data.nodes || data.nodes.length === 0) {
                alert('未找到在线节点');
                return;
            }

            // 随机取10个节点
            const selectedNodes = data.nodes.slice(0, 10);
            const nodeIds = selectedNodes.map(node => node.nodeId).join('\n');
            document.getElementById('nodeIdsInput').value = nodeIds;

        } catch (error) {
            console.error('随机选择节点失败:', error);
            alert(`随机选择节点失败: ${error.message}`);
        }
    }

    // 开始模拟
    async startSimulation() {
        const nodeIdsText = document.getElementById('nodeIdsInput').value.trim();
        if (!nodeIdsText) {
            alert('请输入节点ID');
            return;
        }

        const nodeIds = nodeIdsText.split('\n').map(id => id.trim()).filter(id => id);
        if (nodeIds.length === 0) {
            alert('请输入有效的节点ID');
            return;
        }

        // 停止之前的模拟
        this.stopSimulation();

        // 初始化节点
        this.nodes.clear();
        nodeIds.forEach(nodeId => {
            this.nodes.set(nodeId, {
                nodeId: nodeId,
                status: 'active',
                components: {},
                lastUpdate: null,
                logs: []
            });
        });

        // 更新UI
        this.renderNodes();
        this.updateStatus(true);

        // 立即拉取一次配置
        for (const nodeId of nodeIds) {
            await this.fetchNodeConfig(nodeId);
        }

        // 为每个节点设置定时器（每3秒拉取一次）
        nodeIds.forEach(nodeId => {
            const intervalId = setInterval(() => {
                this.fetchNodeConfig(nodeId);
            }, 3000);
            this.intervalIds.set(nodeId, intervalId);
        });
    }

    // 停止模拟
    stopSimulation() {
        if (!this.isRunning) {
            return;
        }

        // 清除所有定时器
        this.intervalIds.forEach((intervalId) => {
            clearInterval(intervalId);
        });
        this.intervalIds.clear();

        this.updateStatus(false);
    }

    // 获取节点配置
    async fetchNodeConfig(nodeId) {
        try {
            const response = await fetch(`/api/v1/jarvisconf/${nodeId}?devType=jarvis.A`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const config = await response.json();
            
            // 解析组件版本
            const components = this.parseComponents(config);
            
            // 更新节点数据
            const node = this.nodes.get(nodeId);
            if (node) {
                const oldComponents = node.components;
                let hasChanges = false;

                // 检测版本变化
                Object.keys(components).forEach(appName => {
                    const oldVersion = oldComponents[appName];
                    const newVersion = components[appName];
                    if (oldVersion !== newVersion) {
                        hasChanges = true;
                        if (oldVersion) {
                            this.addNodeLog(nodeId, `配置已更新, 共 ${Object.keys(components).length} 个组件【${appName}】 ${oldVersion} -> ${newVersion}`, 'success');
                        } else {
                            this.addNodeLog(nodeId, `配置已更新, 共 ${Object.keys(components).length} 个组件`, 'success');
                        }
                    }
                });

                // 只有首次或有变化时才记录
                if (Object.keys(oldComponents).length === 0) {
                    this.addNodeLog(nodeId, `配置已更新, 共 ${Object.keys(components).length} 个组件`, 'info');
                }

                node.components = components;
                node.lastUpdate = new Date().toLocaleTimeString('zh-CN', { hour12: false });
                node.status = 'active';

                this.renderNodes();
            }

        } catch (error) {
            console.error(`获取节点 ${nodeId} 配置失败:`, error);
            this.addNodeLog(nodeId, `获取配置失败: ${error.message}`, 'error');
            
            const node = this.nodes.get(nodeId);
            if (node) {
                node.status = 'error';
                this.renderNodes();
            }
        }
    }

    // 解析组件版本
    parseComponents(config) {
        const components = {};

        if (!config || !config.apps) {
            return components;
        }

        Object.keys(config.apps).forEach(appName => {
            const app = config.apps[appName];
            if (app && app.main && app.main.url) {
                const version = this.extractVersionFromUrl(app.main.url);
                components[appName] = version;
            }
        });

        return components;
    }

    // 从URL中提取版本号（取url按/分隔后倒数第2个值）
    extractVersionFromUrl(url) {
        if (!url) return '-';
        const parts = url.split('/');
        if (parts.length < 2) return '-';
        return parts[parts.length - 2] || '-';
    }

    // 渲染节点列表
    renderNodes() {
        const container = document.getElementById('nodesContainer');
        
        if (this.nodes.size === 0) {
            container.innerHTML = '<div class="empty-message">请选择节点并开始模拟</div>';
            return;
        }

        container.innerHTML = '';

        this.nodes.forEach((node, nodeId) => {
            const nodeCard = document.createElement('div');
            nodeCard.className = 'node-card';

            const statusClass = node.status === 'active' ? 'active' : 'error';
            const statusText = node.status === 'active' ? '运行中' : '错误';

            let componentsHtml = '';
            if (Object.keys(node.components).length === 0) {
                componentsHtml = '<div class="component-item"><span class="component-name">暂无组件数据</span></div>';
            } else {
                Object.keys(node.components).forEach(appName => {
                    const version = node.components[appName];
                    componentsHtml += `
                        <div class="component-item">
                            <span class="component-name">${appName}</span>
                            <span class="component-version">${version}</span>
                        </div>
                    `;
                });
            }

            nodeCard.innerHTML = `
                <div class="node-header">
                    <span class="node-id">节点ID: ${nodeId}</span>
                    <span class="node-status ${statusClass}">${statusText}</span>
                </div>
                <div class="node-info">
                    最后更新: ${node.lastUpdate || '未更新'}
                </div>
                <div class="node-body">
                    <div class="node-components">
                        ${componentsHtml}
                    </div>
                    <div class="node-logs">
                        <div class="node-logs-header">
                            <span class="node-logs-title">日志</span>
                            <button class="btn btn-sm btn-secondary" onclick="nodeSimulator.clearNodeLogs('${nodeId}')">清空</button>
                        </div>
                        <div class="node-logs-container" id="node-logs-${nodeId}"></div>
                    </div>
                </div>
            `;

            container.appendChild(nodeCard);
        });

        // 更新节点数量
        document.getElementById('nodeCount').textContent = this.nodes.size;
    }

    // 更新状态
    updateStatus(isRunning) {
        this.isRunning = isRunning;
        const statusEl = document.getElementById('simulationStatus');
        const startBtn = document.getElementById('startSimulationBtn');
        const stopBtn = document.getElementById('stopSimulationBtn');
        const randomBtn = document.getElementById('randomSelectBtn');
        const inputEl = document.getElementById('nodeIdsInput');

        if (isRunning) {
            statusEl.textContent = '运行中';
            statusEl.classList.add('running');
            startBtn.disabled = true;
            stopBtn.disabled = false;
            randomBtn.disabled = true;
            inputEl.disabled = true;
        } else {
            statusEl.textContent = '已停止';
            statusEl.classList.remove('running');
            startBtn.disabled = false;
            stopBtn.disabled = true;
            randomBtn.disabled = false;
            inputEl.disabled = false;
        }
    }
}

// 全局实例
let nodeSimulator;

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    nodeSimulator = new NodeSimulator();
});
