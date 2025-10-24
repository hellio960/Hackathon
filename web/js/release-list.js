// 发布任务列表页面JavaScript功能
class ReleaseListManager {
    constructor() {
        this.currentPage = 1;
        this.pageSize = 10;
        this.searchParams = {};
        // 使用代理服务器解决CORS问题
        this.apiBaseUrl = window.location.origin + '/api'; // 通过代理访问后端API
        
        this.init();
    }

    // 初始化
    init() {
        this.bindEvents();
        this.loadData();
    }

    // 绑定事件
    bindEvents() {
        // 搜索按钮点击事件
        document.getElementById('searchBtn').addEventListener('click', () => {
            this.handleSearch();
        });

        // 回车键搜索
        const searchInputs = document.querySelectorAll('.search-input, .search-select');
        searchInputs.forEach(input => {
            input.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleSearch();
                }
            });
        });

        // 分页按钮事件
        document.getElementById('prevPage').addEventListener('click', () => {
            if (this.currentPage > 1) {
                this.currentPage--;
                this.loadData();
            }
        });

        document.getElementById('nextPage').addEventListener('click', () => {
            if (this.currentPage < this.totalPages) {
                this.currentPage++;
                this.loadData();
            }
        });
    }

    // 处理搜索
    handleSearch() {
        this.currentPage = 1;
        this.collectSearchParams();
        this.loadData();
    }

    // 收集搜索参数 - 根据API接口定义NodeReleaseListReq
    collectSearchParams() {
        this.searchParams = {};
        
        // 任务ID - releaseIds (数组)
        const releaseId = document.getElementById('releaseId').value.trim();
        if (releaseId) {
            this.searchParams.releaseIds = [releaseId];
        }

        // 设备类型 - devTypes (数组)
        const deviceType = document.getElementById('deviceType').value;
        if (deviceType) {
            this.searchParams.devTypes = [deviceType];
        }

        // 发布类型 - releaseTypes (数组)
        const releaseType = document.getElementById('releaseType').value;
        if (releaseType) {
            this.searchParams.releaseTypes = [releaseType];
        }

        // 应用名 - app (字符串)
        const appName = document.getElementById('appName').value.trim();
        if (appName) {
            this.searchParams.app = appName;
        }

        // 发布状态 - states (数组)
        const state = document.getElementById('state').value;
        if (state) {
            this.searchParams.states = [state];
        }
    }

    // 加载数据
    async loadData() {
        try {
            this.showLoading(true);
            
            // 根据API接口定义构建请求数据 - NodeReleaseListReq
            // 实际后端使用小写字段名
            const requestData = {
                page: this.currentPage,
                size: this.pageSize
            };

            // 添加可选的搜索参数
            if (this.searchParams.releaseIds) {
                requestData.releaseIds = this.searchParams.releaseIds;
            }
            if (this.searchParams.app) {
                requestData.app = this.searchParams.app;
            }
            if (this.searchParams.devTypes) {
                requestData.devTypes = this.searchParams.devTypes;
            }
            if (this.searchParams.releaseTypes) {
                requestData.releaseTypes = this.searchParams.releaseTypes;
            }
            if (this.searchParams.states) {
                requestData.states = this.searchParams.states;
            }

            console.log('发送请求数据:', requestData);

            const response = await fetch(`${this.apiBaseUrl}/v1/release/list`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestData),
                mode: 'cors', // 明确指定CORS模式
                credentials: 'omit' // 不发送凭据
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            console.log('接收响应数据:', data);
            
            // 检查是否有错误码 - 只有当存在code字段且不为0时才认为是错误
            if (data.code !== undefined && data.code !== 0) {
                throw new Error(data.desc || data.message || '请求失败');
            }
            
            // 根据API接口定义NodeReleaseListResp处理响应
            // 后端返回格式: {"items":null,"total":0} 或 {"items":[...],"total":N}
            if (data !== null && typeof data === 'object' && data.hasOwnProperty('items') && data.hasOwnProperty('total')) {
                const items = data.items || [];
                const total = data.total || 0;
                this.renderTable(items);
                this.updatePagination(total);
            } else {
                throw new Error('响应数据格式不正确');
            }
        } catch (error) {
            console.error('加载数据失败:', error);
            
            // 如果是网络错误，提供更友好的错误信息
            let errorMessage = error.message;
            if (error.name === 'TypeError' && error.message.includes('fetch')) {
                errorMessage = '无法连接到服务器，请检查：\n1. 后端服务是否正常运行\n2. 服务器是否支持跨域请求(CORS)\n3. 网络连接是否正常';
            }
            
            this.showError('加载数据失败: ' + errorMessage);
            this.renderTable([]);
            this.updatePagination(0);
        } finally {
            this.showLoading(false);
        }
    }

    // 渲染表格 - 根据API接口定义NodeReleaseBrief
    renderTable(items) {
        const tbody = document.getElementById('tableBody');
        
        if (!items || items.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="8" class="empty-state">
                        <div class="empty-state-icon">📋</div>
                        <div class="empty-state-text">暂无数据</div>
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = items.map(item => `
            <tr>
                <td>${this.escapeHtml(item.id || '')}</td>
                <td>${this.escapeHtml(item.app || '')}</td>
                <td>${this.escapeHtml(item.deviceType || '')}</td>
                <td>${this.getReleaseTypeText(item.releaseType)}</td>
                <td>${this.getStatusTag(item.state)}</td>
                <td>${this.escapeHtml(item.operator || '')}</td>
                <td>${this.formatDateTime(item.createAt)}</td>
                <td>
                    <a href="#" class="action-btn action-btn-detail" onclick="releaseManager.viewDetail('${item.id}')">详情</a>
                </td>
            </tr>
        `).join('');
    }

    // 获取发布类型文本
    getReleaseTypeText(type) {
        const typeMap = {
            'formal': '正式发布',
            'beta': '功能验证'
        };
        return typeMap[type] || type || '';
    }

    // 获取状态标签
    getStatusTag(state) {
        const stateMap = {
            'created': { text: '已创建', class: 'status-created' },
            'running': { text: '进行中', class: 'status-running' },
            'completed': { text: '完成', class: 'status-completed' },
            'paused': { text: '暂停', class: 'status-paused' },
            'rollback': { text: '回滚', class: 'status-rollback' }
        };
        
        const stateInfo = stateMap[state] || { text: state || '', class: 'status-created' };
        return `<span class="status-tag ${stateInfo.class}">${stateInfo.text}</span>`;
    }

    // 格式化日期时间 - createAt是int64类型的时间戳
    formatDateTime(timestamp) {
        if (!timestamp) return '';
        
        // 根据API定义，createAt是int64类型的时间戳（秒）
        const date = new Date(timestamp * 1000);
        
        // 检查日期是否有效
        if (isNaN(date.getTime())) {
            return '';
        }
        
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const seconds = String(date.getSeconds()).padStart(2, '0');
        
        return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
    }

    // 更新分页信息
    updatePagination(total) {
        this.totalCount = total;
        this.totalPages = Math.ceil(total / this.pageSize);
        
        // 更新总数显示
        document.getElementById('totalCount').textContent = total;
        document.getElementById('totalPages').textContent = this.totalPages;
        
        // 更新分页按钮状态
        const prevBtn = document.getElementById('prevPage');
        const nextBtn = document.getElementById('nextPage');
        
        prevBtn.disabled = this.currentPage <= 1;
        nextBtn.disabled = this.currentPage >= this.totalPages;
        
        // 生成页码
        this.renderPageNumbers();
    }

    // 渲染页码
    renderPageNumbers() {
        const pageNumbersContainer = document.getElementById('pageNumbers');
        const maxVisiblePages = 5;
        let startPage = Math.max(1, this.currentPage - Math.floor(maxVisiblePages / 2));
        let endPage = Math.min(this.totalPages, startPage + maxVisiblePages - 1);
        
        if (endPage - startPage + 1 < maxVisiblePages) {
            startPage = Math.max(1, endPage - maxVisiblePages + 1);
        }
        
        let html = '';
        
        // 第一页
        if (startPage > 1) {
            html += `<div class="page-number" onclick="releaseManager.goToPage(1)">1</div>`;
            if (startPage > 2) {
                html += `<div class="page-number">...</div>`;
            }
        }
        
        // 中间页码
        for (let i = startPage; i <= endPage; i++) {
            const activeClass = i === this.currentPage ? 'active' : '';
            html += `<div class="page-number ${activeClass}" onclick="releaseManager.goToPage(${i})">${i}</div>`;
        }
        
        // 最后一页
        if (endPage < this.totalPages) {
            if (endPage < this.totalPages - 1) {
                html += `<div class="page-number">...</div>`;
            }
            html += `<div class="page-number" onclick="releaseManager.goToPage(${this.totalPages})">${this.totalPages}</div>`;
        }
        
        pageNumbersContainer.innerHTML = html;
    }

    // 跳转到指定页
    goToPage(page) {
        if (page >= 1 && page <= this.totalPages && page !== this.currentPage) {
            this.currentPage = page;
            this.loadData();
        }
    }

    // 查看详情
    viewDetail(releaseId) {
        // 这里可以跳转到详情页面或打开详情弹窗
        console.log('查看详情:', releaseId);
        alert(`查看任务详情: ${releaseId}`);
    }

    // 显示/隐藏加载状态
    showLoading(show) {
        const loading = document.getElementById('loading');
        if (show) {
            loading.classList.remove('hidden');
        } else {
            loading.classList.add('hidden');
        }
    }

    // 显示错误信息
    showError(message) {
        alert('错误: ' + message);
    }

    // HTML转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    window.releaseManager = new ReleaseListManager();
});

// 导出到全局作用域，供HTML中的onclick使用
window.releaseManager = null;