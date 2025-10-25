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
            'processing': { text: '进行中', class: 'status-running' },
            'complete': { text: '完成', class: 'status-completed' },
            'rollbacked': { text: '回滚', class: 'status-rollback' }
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
        // 跳转到详情页面
        window.location.href = `release-detail.html?id=${releaseId}`;
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
        this.showToast(message, 'error');
    }

    // 显示成功信息
    showSuccess(message) {
        this.showToast(message, 'success');
    }

    // 显示Toast通知
    showToast(message, type = 'success') {
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = 'toast ' + type;
        
        // 触发显示动画
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);
        
        // 3秒后自动隐藏
        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }

    // HTML转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// 创建发布任务管理器
class ReleaseCreateManager {
    constructor(listManager) {
        this.listManager = listManager;
        this.currentStep = 1;
        this.formData = {};
        this.currentConfig = null;
        this.packagesList = [];
        this.customPath = null;
        
        this.bindEvents();
    }

    bindEvents() {
        // 显示创建弹窗按钮
        document.getElementById('showCreateModalBtn').addEventListener('click', () => {
            this.showCreateModal();
        });

        // 关闭弹窗
        document.getElementById('closeCreateModal').addEventListener('click', () => {
            this.hideCreateModal();
        });

        document.getElementById('createCancelBtn').addEventListener('click', () => {
            this.hideCreateModal();
        });

        // 步骤按钮
        document.getElementById('createNextBtn').addEventListener('click', () => {
            this.handleNextStep();
        });

        document.getElementById('createPrevBtn').addEventListener('click', () => {
            this.handlePrevStep();
        });

        document.getElementById('createSubmitBtn').addEventListener('click', () => {
            this.handleSubmit();
        });

        // 组件和设备类型变化
        document.getElementById('createAppName').addEventListener('change', () => {
            this.handleAppOrDevTypeChange();
        });

        document.getElementById('createDevType').addEventListener('change', () => {
            this.handleAppOrDevTypeChange();
        });

        // 操作类型变化
        document.getElementById('createOpType').addEventListener('change', (e) => {
            this.handleOpTypeChange(e.target.value);
        });

        // 灰度模式切换
        document.querySelectorAll('input[name="createGrayMode"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.toggleGrayMode(e.target.value);
            });
        });

        // 包选择变化
        document.getElementById('createPackageSelect').addEventListener('change', (e) => {
            this.handlePackageChange(e.target.value);
        });

        // 指定路径
        document.getElementById('specifyPathBtn').addEventListener('click', () => {
            document.getElementById('pathFilterModal').classList.remove('hidden');
        });

        document.getElementById('closePathFilterModal').addEventListener('click', () => {
            document.getElementById('pathFilterModal').classList.add('hidden');
        });

        document.getElementById('cancelPathFilterBtn').addEventListener('click', () => {
            document.getElementById('pathFilterModal').classList.add('hidden');
        });

        document.getElementById('confirmPathFilterBtn').addEventListener('click', () => {
            this.handlePathFilter();
        });

        // 点击弹窗外部关闭
        document.getElementById('createReleaseModal').addEventListener('click', (e) => {
            if (e.target.id === 'createReleaseModal') {
                this.hideCreateModal();
            }
        });

        document.getElementById('pathFilterModal').addEventListener('click', (e) => {
            if (e.target.id === 'pathFilterModal') {
                document.getElementById('pathFilterModal').classList.add('hidden');
            }
        });

        // 初始化节点阶段和业务ID自定义多选下拉框
        this.initCustomMultiSelect('createStagesWrapper');
        this.initCustomMultiSelect('createCustomerIdsWrapper');
        
        // 点击页面其他地方关闭下拉框
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.custom-multiselect')) {
                document.querySelectorAll('.custom-multiselect.active').forEach(el => {
                    el.classList.remove('active');
                });
            }
        });
    }

    // 初始化自定义多选下拉框
    initCustomMultiSelect(wrapperId) {
        const wrapper = document.getElementById(wrapperId);
        if (!wrapper) return;

        const trigger = wrapper.querySelector('.multiselect-trigger');
        const textEl = wrapper.querySelector('.multiselect-text');
        const selectAllCheckbox = wrapper.querySelector('.select-all-checkbox');
        const optionCheckboxes = wrapper.querySelectorAll('.multiselect-options input[type="checkbox"]');

        // 点击触发器展开/收起下拉框
        trigger.addEventListener('click', (e) => {
            e.stopPropagation();
            wrapper.classList.toggle('active');
        });

        // 全选/取消全选
        selectAllCheckbox.addEventListener('change', () => {
            const checked = selectAllCheckbox.checked;
            optionCheckboxes.forEach(checkbox => {
                checkbox.checked = checked;
            });
            this.updateMultiSelectText(wrapper);
        });

        // 选项变化
        optionCheckboxes.forEach(checkbox => {
            checkbox.addEventListener('change', () => {
                this.updateMultiSelectText(wrapper);
                this.updateSelectAllState(wrapper);
            });
        });

        // 初始化显示文本
        this.updateMultiSelectText(wrapper);
    }

    // 更新多选下拉框显示文本
    updateMultiSelectText(wrapper) {
        const textEl = wrapper.querySelector('.multiselect-text');
        const checkboxes = wrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked');
        
        if (checkboxes.length === 0) {
            textEl.textContent = '请选择';
            textEl.classList.add('placeholder');
        } else {
            const labels = Array.from(checkboxes).map(cb => {
                return cb.parentElement.querySelector('span').textContent;
            });
            textEl.textContent = labels.join(', ');
            textEl.classList.remove('placeholder');
        }
    }

    // 更新全选按钮状态
    updateSelectAllState(wrapper) {
        const selectAllCheckbox = wrapper.querySelector('.select-all-checkbox');
        const optionCheckboxes = wrapper.querySelectorAll('.multiselect-options input[type="checkbox"]');
        const checkedCount = wrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked').length;
        
        selectAllCheckbox.checked = checkedCount === optionCheckboxes.length;
    }

    // 显示创建弹窗
    async showCreateModal() {
        this.resetForm();
        await this.loadComponents();
        document.getElementById('createReleaseModal').classList.remove('hidden');
    }

    // 隐藏创建弹窗
    hideCreateModal() {
        document.getElementById('createReleaseModal').classList.add('hidden');
        this.resetForm();
    }

    // 重置表单
    resetForm() {
        this.currentStep = 1;
        this.formData = {};
        this.currentConfig = null;
        this.packagesList = [];
        this.customPath = null;

        // 显示步骤1
        document.getElementById('createStep1').classList.add('active');
        document.getElementById('createStep2').classList.remove('active');

        // 按钮状态
        document.getElementById('createPrevBtn').style.display = 'none';
        document.getElementById('createNextBtn').style.display = 'inline-block';
        document.getElementById('createSubmitBtn').style.display = 'none';

        // 清空表单
        document.getElementById('createAppName').value = '';
        document.getElementById('createDevType').value = '';
        document.getElementById('createOpType').value = '';
        document.querySelector('input[name="createReleaseType"][value="formal"]').checked = true;
        document.getElementById('createPackageSelect').innerHTML = '<option value="">请选择版本包</option>';
        document.getElementById('createPackageType').value = '';
        document.getElementById('createCmd').value = '';
        document.getElementById('createWorkDir').value = '';
        document.getElementById('createArgs').value = '';
        document.getElementById('createNodeIds').value = '';
        document.getElementById('createNodeStatus').value = 'online';
        
        // 清空节点阶段多选
        const stagesWrapper = document.getElementById('createStagesWrapper');
        stagesWrapper.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
        this.updateMultiSelectText(stagesWrapper);
        
        // 清空业务ID多选
        const customerIdsWrapper = document.getElementById('createCustomerIdsWrapper');
        customerIdsWrapper.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
        this.updateMultiSelectText(customerIdsWrapper);
        
        document.getElementById('createPercentage').value = '10';
        document.getElementById('createDescribe').value = '';
        document.querySelector('input[name="createGrayMode"][value="nodeIds"]').checked = true;
        this.toggleGrayMode('nodeIds');
    }

    // 加载组件列表
    async loadComponents() {
        try {
            const response = await fetch('/api/v1/release/allowapps', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            
            if (data.code && data.code !== 0) {
                throw new Error(data.desc || '获取组件列表失败');
            }

            this.renderComponentOptions(data.apps || []);

        } catch (error) {
            console.error('加载组件列表失败:', error);
            this.showError('加载组件列表失败: ' + error.message);
        }
    }

    // 渲染组件选项
    renderComponentOptions(apps) {
        const select = document.getElementById('createAppName');
        select.innerHTML = '<option value="">请选择组件</option>';
        
        const uniqueApps = [...new Set(apps.map(app => app.name))];
        
        uniqueApps.forEach(appName => {
            const option = document.createElement('option');
            option.value = appName;
            option.textContent = appName;
            select.appendChild(option);
        });
    }

    // 处理组件或设备类型变化
    async handleAppOrDevTypeChange() {
        const appName = document.getElementById('createAppName').value;
        const devType = document.getElementById('createDevType').value;

        if (!appName || !devType) {
            return;
        }

        // 先加载当前配置，然后根据结果决定是否加载包列表
        await this.loadCurrentConfig(appName, devType);
        
        // 不在这里加载包列表，在点击"下一步"时根据操作类型判断后再加载
    }

    // 加载当前配置
    async loadCurrentConfig(appName, devType) {
        try {
            // devType现在已经是正确的格式（如 jarvis.A, ant.A 等），直接使用
            const url = `/api/v1/jarvisconf?devType=${devType}`;
            
            console.log('请求配置URL:', url); // 调试信息
            
            const response = await fetch(url, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            console.log('响应状态:', response.status, response.statusText); // 调试信息

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status} ${response.statusText}`);
            }

            // 获取响应文本，然后解析
            const responseText = await response.text();
            console.log('响应文本:', responseText); // 调试信息
            
            // 检查响应是否为空
            if (!responseText || responseText.trim() === '') {
                console.log('响应为空，认为配置不存在');
                this.currentConfig = null;
                return;
            }

            // 尝试解析JSON
            let data;
            try {
                data = JSON.parse(responseText);
                console.log('解析后的数据:', data); // 调试信息
            } catch (parseError) {
                console.error('JSON解析失败:', parseError, '响应内容:', responseText);
                throw new Error('服务器返回的数据格式错误');
            }

            // 注意：/v1/jarvisconf 接口直接返回 JarvisUpdConf 结构，没有 code 字段
            // JarvisUpdConf 结构: { name, apps: {appName: {main, alter}}, ttl, ... }
            
            // 判断当前组件配置是否存在
            // data.apps 是一个 map，key 是应用名，value 包含 main 和 alter
            if (data && data.apps && data.apps[appName] && data.apps[appName].main) {
                this.currentConfig = data.apps[appName].main;
                console.log(`✅ 组件 "${appName}" 存在当前配置`, this.currentConfig); // 调试信息
            } else {
                this.currentConfig = null;
                console.log(`ℹ️ 组件 "${appName}" 不存在当前配置`); // 调试信息
            }

        } catch (error) {
            console.error('❌ 加载当前配置失败:', error);
            this.currentConfig = null;
            
            // 根据错误类型给出不同提示
            let errorMessage = '获取当前配置失败';
            if (error.message.includes('Failed to fetch')) {
                errorMessage = '无法连接到服务器，请检查网络连接或后端服务是否启动';
            } else if (error.message.includes('HTTP error')) {
                errorMessage = `服务器错误: ${error.message}`;
            } else {
                errorMessage = error.message;
            }
            
            this.showError(errorMessage);
        }
    }

    // 加载包列表
    async loadPackages(appName, devType, path = null) {
        try {
            // 不显示全局loading，因为这是后台异步加载
            // this.listManager.showLoading(true);

            const requestData = {
                app: appName,
                devType: devType,
                size: 20
            };

            if (path) {
                requestData.path = path;
            }

            console.log('请求包列表:', requestData); // 调试信息

            // 使用带重试和超时的fetch函数（超时60秒，最多重试10次）
            const response = await fetchWithRetry('/api/v1/release/packages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            }, 10, 60000);

            console.log('包列表响应状态:', response.status, response.statusText); // 调试信息

            if (!response.ok) {
                // 不抛出异常，静默处理
                console.warn(`⚠️ 获取包列表失败: HTTP ${response.status} ${response.statusText}`);
                const errorText = await response.text();
                console.warn('错误详情:', errorText);
                this.packagesList = [];
                this.renderPackageOptions();
                return;
            }

            const responseText = await response.text();
            
            // 检查响应是否为空
            if (!responseText || responseText.trim() === '') {
                console.log('ℹ️ 包列表响应为空');
                this.packagesList = [];
                this.renderPackageOptions();
                return;
            }

            let data;
            try {
                data = JSON.parse(responseText);
                console.log('包列表数据:', data); // 调试信息
            } catch (parseError) {
                console.error('❌ JSON解析失败:', parseError);
                this.packagesList = [];
                this.renderPackageOptions();
                return;
            }
            
            if (data.code && data.code !== 0) {
                console.warn(`⚠️ 获取包列表失败: ${data.desc || data.message || '未知错误'}`);
                this.packagesList = [];
                this.renderPackageOptions();
                return;
            }

            this.packagesList = data.packages || [];
            console.log(`✅ 获取到 ${this.packagesList.length} 个包`);
            this.renderPackageOptions();

        } catch (error) {
            // 捕获网络错误等异常，静默处理
            console.error('❌ 加载包列表异常:', error);
            this.packagesList = [];
            this.renderPackageOptions();
        } finally {
            // 不需要关闭全局loading
            // this.listManager.showLoading(false);
        }
    }

    // 渲染包选项
    renderPackageOptions() {
        const select = document.getElementById('createPackageSelect');
        select.innerHTML = '<option value="">请选择版本包</option>';
        
        this.packagesList.forEach((pkg, index) => {
            const option = document.createElement('option');
            option.value = index;
            option.textContent = `${pkg.file} (${pkg.size})`;
            select.appendChild(option);
        });
    }

    // 处理包选择变化
    handlePackageChange(value) {
        if (!value) return;

        const pkg = this.packagesList[parseInt(value)];
        if (!pkg) return;

        document.getElementById('createCmd').value = this.extractProgramName(pkg.file);
        
        const packageType = this.guessPackageType(pkg.file);
        document.getElementById('createPackageType').value = packageType;
    }

    // 从文件名提取程序名
    extractProgramName(filename) {
        return filename.replace(/.*\//, '').replace(/\.(tar\.gz|tar|zip|tgz)$/, '');
    }

    // 推测包类型
    guessPackageType(filename) {
        if (filename.match(/\.(tar\.gz|tar|zip|tgz)$/)) {
            return 'zipped';
        }
        return 'executable';
    }

    // 操作类型变化
    handleOpTypeChange(opType) {
        // 如果已经选择了组件和设备类型，并且已经加载了配置，给出提示
        const appName = document.getElementById('createAppName').value;
        const devType = document.getElementById('createDevType').value;
        
        if (appName && devType && this.currentConfig !== undefined) {
            const hasCurrentConfig = this.currentConfig !== null;
            
            // 根据操作类型和配置存在情况给出提示
            if (opType === 'add' && hasCurrentConfig) {
                console.warn(`组件 "${appName}" 已存在配置，新增操作可能失败`);
            } else if (opType === 'update' && !hasCurrentConfig) {
                console.warn(`组件 "${appName}" 不存在配置，升级操作可能失败`);
            } else if (opType === 'delete' && !hasCurrentConfig) {
                console.warn(`组件 "${appName}" 不存在配置，移除操作可能失败`);
            }
        }
    }

    // 切换灰度模式
    toggleGrayMode(mode) {
        const nodeIdsSection = document.getElementById('createNodeIdsSection');
        const filterSection = document.getElementById('createFilterSection');

        if (mode === 'nodeIds') {
            nodeIdsSection.style.display = 'block';
            filterSection.style.display = 'none';
        } else {
            nodeIdsSection.style.display = 'none';
            filterSection.style.display = 'block';
        }
    }

    // 处理路径筛选
    handlePathFilter() {
        const path = document.getElementById('customPath').value.trim();
        
        if (!path) {
            this.showError('请输入包路径');
            return;
        }

        const appName = this.formData.appName;
        const devType = this.formData.devType;

        this.customPath = path;
        document.getElementById('pathFilterModal').classList.add('hidden');
        document.getElementById('customPath').value = '';

        // 异步加载包列表
        const select = document.getElementById('createPackageSelect');
        select.innerHTML = '<option value="">加载中...</option>';
        select.disabled = true;
        
        this.loadPackages(appName, devType, path).then(() => {
            select.disabled = false;
        }).catch(error => {
            select.disabled = false;
        });
    }

    // 下一步
    handleNextStep() {
        if (!this.validateStep1()) {
            return;
        }

        // 保存步骤1数据
        this.formData = {
            appName: document.getElementById('createAppName').value,
            devType: document.getElementById('createDevType').value,
            releaseType: document.querySelector('input[name="createReleaseType"]:checked').value,
            opType: document.getElementById('createOpType').value
        };

        // 验证操作类型与当前配置的一致性
        const opType = this.formData.opType;
        const hasCurrentConfig = this.currentConfig !== null;

        if (opType === 'add') {
            // 新增组件：要求当前配置不存在
            if (hasCurrentConfig) {
                this.showError(`组件 "${this.formData.appName}" 已存在配置，不能执行新增操作。请选择"升级"操作。`);
                return;
            }
        } else if (opType === 'update') {
            // 升级组件：要求当前配置存在
            if (!hasCurrentConfig) {
                this.showError(`组件 "${this.formData.appName}" 不存在当前配置，不能执行升级操作。请选择"新增"操作。`);
                return;
            }
        } else if (opType === 'delete') {
            // 移除组件：要求当前配置存在
            if (!hasCurrentConfig) {
                this.showError(`组件 "${this.formData.appName}" 不存在当前配置，不能执行移除操作。`);
                return;
            }
        }

        // 显示步骤2（立即切换，不等待包列表加载）
        document.getElementById('createStep1').classList.remove('active');
        document.getElementById('createStep2').classList.add('active');
        this.currentStep = 2;

        // 异步加载包列表（移除操作不需要）
        if (opType !== 'delete') {
            // 先显示加载中状态
            const select = document.getElementById('createPackageSelect');
            select.innerHTML = '<option value="">加载中...</option>';
            select.disabled = true;
            
            // 异步加载包列表
            this.loadPackages(this.formData.appName, this.formData.devType).then(() => {
                // 加载完成后启用下拉框
                select.disabled = false;
            }).catch(error => {
                // 错误已在 loadPackages 中处理
                select.disabled = false;
            });
        }

        // 按钮状态
        document.getElementById('createPrevBtn').style.display = 'inline-block';
        document.getElementById('createNextBtn').style.display = 'none';
        document.getElementById('createSubmitBtn').style.display = 'inline-block';

        // 填充信息回显
        document.getElementById('recapAppName').textContent = this.formData.appName;
        document.getElementById('recapDevType').textContent = this.getDevTypeText(this.formData.devType);
        document.getElementById('recapReleaseType').textContent = this.getReleaseTypeText(this.formData.releaseType);
        document.getElementById('recapOpType').textContent = this.getOpTypeText(this.formData.opType);

        // 填充当前版本配置
        this.displayCurrentConfig();

        // 如果存在当前配置，预填充发布版本字段
        this.prefillReleaseFields();
    }

    // 上一步
    handlePrevStep() {
        document.getElementById('createStep2').classList.remove('active');
        document.getElementById('createStep1').classList.add('active');
        this.currentStep = 1;

        // 按钮状态
        document.getElementById('createPrevBtn').style.display = 'none';
        document.getElementById('createNextBtn').style.display = 'inline-block';
        document.getElementById('createSubmitBtn').style.display = 'none';
    }

    // 验证步骤1
    validateStep1() {
        const appName = document.getElementById('createAppName').value;
        const devType = document.getElementById('createDevType').value;
        const opType = document.getElementById('createOpType').value;

        if (!appName) {
            this.showError('请选择组件');
            return false;
        }

        if (!devType) {
            this.showError('请选择设备类型');
            return false;
        }

        if (!opType) {
            this.showError('请选择操作类型');
            return false;
        }

        return true;
    }

    // 显示当前版本配置
    displayCurrentConfig() {
        const opType = this.formData.opType;
        const currentVersionSection = document.getElementById('currentVersionSection');
        
        if (opType === 'add') {
            // 新增操作：不显示当前版本区域
            currentVersionSection.style.display = 'none';
        } else if (opType === 'update' || opType === 'delete') {
            // 升级或移除操作：显示当前版本配置
            currentVersionSection.style.display = 'block';
            
            if (this.currentConfig) {
                // 填充当前版本配置
                // 注意：API返回的字段名（根据json标签）是小写的
                document.getElementById('currentUrl').textContent = this.currentConfig.url || '-';
                document.getElementById('currentMd5').textContent = this.currentConfig.md5 || '-';
                document.getElementById('currentPackageType').textContent = this.currentConfig.type || '-';
                document.getElementById('currentCmd').textContent = this.currentConfig.cmd || '-';
                document.getElementById('currentWorkDir').textContent = this.currentConfig.dir || '-';
                
                console.log('当前配置已填充:', {
                    url: this.currentConfig.url,
                    md5: this.currentConfig.md5,
                    type: this.currentConfig.type,
                    cmd: this.currentConfig.cmd,
                    dir: this.currentConfig.dir
                });
            } else {
                // 理论上不应该走到这里，因为在handleNextStep已经验证了
                document.getElementById('currentUrl').textContent = '-';
                document.getElementById('currentMd5').textContent = '-';
                document.getElementById('currentPackageType').textContent = '-';
                document.getElementById('currentCmd').textContent = '-';
                document.getElementById('currentWorkDir').textContent = '-';
            }
        }
    }

    // 预填充发布版本字段（基于当前配置）
    prefillReleaseFields() {
        const opType = this.formData.opType;

        // 只在升级或删除操作时，且存在当前配置时预填充
        if ((opType === 'update' || opType === 'delete') && this.currentConfig) {
            console.log('预填充发布版本字段:', this.currentConfig);

            // 包类型
            if (this.currentConfig.type) {
                document.getElementById('createPackageType').value = this.currentConfig.type;
            }

            // 程序名
            if (this.currentConfig.cmd) {
                document.getElementById('createCmd').value = this.currentConfig.cmd;
            }

            // 工作目录
            if (this.currentConfig.dir) {
                document.getElementById('createWorkDir').value = this.currentConfig.dir;
            }

            // 启动参数（数组转为空格分隔的字符串）
            if (this.currentConfig.args && Array.isArray(this.currentConfig.args)) {
                document.getElementById('createArgs').value = this.currentConfig.args.join(' ');
            } else if (this.currentConfig.args) {
                // 如果不是数组，直接设置
                document.getElementById('createArgs').value = this.currentConfig.args;
            }

            console.log('✅ 发布版本字段预填充完成');
        } else if (opType === 'add') {
            // 新增操作，清空这些字段
            document.getElementById('createPackageType').value = '';
            document.getElementById('createCmd').value = '';
            document.getElementById('createWorkDir').value = '';
            document.getElementById('createArgs').value = '';
        }
    }

    // 提交表单
    async handleSubmit() {
        if (!this.validateStep2()) {
            return;
        }

        const requestData = this.buildRequestData();

        try {
            this.listManager.showLoading(true);

            console.log('提交创建发布任务:', requestData); // 调试信息

            const response = await fetch('/api/v1/release/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            console.log('创建响应状态:', response.status, response.statusText); // 调试信息

            // 读取响应文本
            const responseText = await response.text();
            console.log('创建响应内容:', responseText || '(空响应)'); // 调试信息

            if (!response.ok) {
                // 尝试解析错误信息
                if (responseText) {
                    try {
                        const errorData = JSON.parse(responseText);
                        throw new Error(errorData.desc || errorData.message || `HTTP ${response.status}`);
                    } catch (parseError) {
                        throw new Error(`HTTP ${response.status}: ${responseText}`);
                    }
                } else {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
            }

            // 成功响应：create接口可能返回空响应
            if (responseText && responseText.trim() !== '') {
                try {
                    const data = JSON.parse(responseText);
                    if (data.code && data.code !== 0) {
                        throw new Error(data.desc || '创建发布任务失败');
                    }
                } catch (parseError) {
                    console.warn('解析响应数据失败，但请求已成功:', parseError);
                    // 继续执行，因为create接口返回空响应时也是成功的
                }
            }

            console.log('✅ 创建发布任务成功');
            this.showSuccess('创建发布任务成功');
            this.hideCreateModal();
            
            // 重新加载列表
            this.listManager.loadData();

        } catch (error) {
            console.error('❌ 创建发布任务失败:', error);
            this.showError('创建发布任务失败: ' + error.message);
        } finally {
            this.listManager.showLoading(false);
        }
    }

    // 验证步骤2
    validateStep2() {
        const opType = this.formData.opType;
        
        if (opType !== 'delete') {
            const packageSelect = document.getElementById('createPackageSelect').value;
            const packageType = document.getElementById('createPackageType').value;
            const cmd = document.getElementById('createCmd').value.trim();
            const workDir = document.getElementById('createWorkDir').value.trim();

            if (!packageSelect) {
                this.showError('请选择版本包');
                return false;
            }

            if (!packageType) {
                this.showError('请选择包类型');
                return false;
            }

            if (!cmd) {
                this.showError('请输入程序名');
                return false;
            }

            if (!workDir) {
                this.showError('请输入工作目录');
                return false;
            }
        }

        const grayMode = document.querySelector('input[name="createGrayMode"]:checked').value;
        
        if (grayMode === 'nodeIds') {
            const nodeIds = document.getElementById('createNodeIds').value.trim();
            if (!nodeIds) {
                this.showError('请输入灰度节点');
                return false;
            }
        }

        const percentage = document.getElementById('createPercentage').value;
        if (!percentage || percentage < 0 || percentage > 100) {
            this.showError('请输入有效的灰度比例（0-100）');
            return false;
        }

        const describe = document.getElementById('createDescribe').value.trim();
        if (!describe) {
            this.showError('请输入发布描述');
            return false;
        }

        return true;
    }

    // 构造请求数据
    buildRequestData() {
        const opType = this.formData.opType;
        
        const requestData = {
            devType: this.formData.devType,
            appName: this.formData.appName,
            describe: document.getElementById('createDescribe').value.trim(),
            releaseType: this.formData.releaseType,
            opType: opType,
            grayPolicy: this.buildGrayPolicy()
        };

        if (opType !== 'delete') {
            requestData.appConfig = this.buildAppConfig();
        }

        return requestData;
    }

    // 构造应用配置
    buildAppConfig() {
        const packageIndex = parseInt(document.getElementById('createPackageSelect').value);
        const pkg = this.packagesList[packageIndex];
        
        const args = document.getElementById('createArgs').value.trim();
        const argsArray = args ? args.split(/\s+/) : [];

        return {
            url: pkg.url,
            type: document.getElementById('createPackageType').value,
            cmd: document.getElementById('createCmd').value.trim(),
            args: argsArray,
            dir: document.getElementById('createWorkDir').value.trim(),
            healthUrl: '',
            md5: pkg.md5
        };
    }

    // 构造灰度策略
    buildGrayPolicy() {
        const grayMode = document.querySelector('input[name="createGrayMode"]:checked').value;
        const percentage = parseInt(document.getElementById('createPercentage').value);
        
        const policy = {
            percentage: percentage,
            statInfo: {
                totalCount: 0
            }
        };

        if (grayMode === 'nodeIds') {
            const nodeIdsText = document.getElementById('createNodeIds').value.trim();
            policy.nodeIds = nodeIdsText.split('\n').map(id => id.trim()).filter(id => id);
        } else {
            const filter = {};
            
            // 节点状态
            const status = document.getElementById('createNodeStatus').value;
            if (status) {
                filter.status = status;
            }
            
            // 节点阶段
            const stagesWrapper = document.getElementById('createStagesWrapper');
            const stagesCheckboxes = stagesWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked');
            const stages = Array.from(stagesCheckboxes).map(cb => cb.value);
            if (stages.length > 0) {
                filter.stages = stages;
            }
            
            // 业务ID
            const customerIdsWrapper = document.getElementById('createCustomerIdsWrapper');
            const customerIdsCheckboxes = customerIdsWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked');
            const customerIds = Array.from(customerIdsCheckboxes).map(cb => parseInt(cb.value));
            if (customerIds.length > 0) {
                filter.customerIds = customerIds;
            }
            
            // 设备类型
            filter.devType = this.formData.devType;
            
            policy.filter = filter;
        }

        return policy;
    }

    // 获取设备类型文本
    getDevTypeText(devType) {
        const map = {
            'jarvis.A': 'jarvis.A',
            'ant.A': 'ant.A',
            'ant.B': 'ant.B',
            'ant.C': 'ant.C'
        };
        return map[devType] || devType;
    }

    // 获取发布类型文本
    getReleaseTypeText(releaseType) {
        const map = {
            'formal': '正式发布',
            'beta': '功能验证'
        };
        return map[releaseType] || releaseType;
    }

    // 获取操作类型文本
    getOpTypeText(opType) {
        const map = {
            'add': '新增',
            'update': '升级',
            'delete': '移除'
        };
        return map[opType] || opType;
    }

    // 显示错误信息
    showError(message) {
        this.showToast(message, 'error');
    }

    // 显示成功信息
    showSuccess(message) {
        this.showToast(message, 'success');
    }

    // 显示Toast通知
    showToast(message, type = 'success') {
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = 'toast ' + type;
        
        // 触发显示动画
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);
        
        // 3秒后自动隐藏
        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }
}

// ========== 工具函数 ==========
// 带重试和超时的fetch函数
async function fetchWithRetry(url, options = {}, maxRetries = 10, timeout = 60000) {
    for (let i = 0; i < maxRetries; i++) {
        try {
            console.log(`[fetchWithRetry] 第${i + 1}/${maxRetries}次请求: ${url}`);
            
            // 创建超时控制
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), timeout);
            
            const response = await fetch(url, {
                ...options,
                signal: controller.signal
            });
            
            clearTimeout(timeoutId);
            
            // 如果成功，返回response
            if (response.ok) {
                console.log(`[fetchWithRetry] 请求成功: ${url}`);
                return response;
            }
            
            // 如果是客户端错误（4xx），不重试
            if (response.status >= 400 && response.status < 500) {
                console.warn(`[fetchWithRetry] 客户端错误 ${response.status}, 不重试`);
                return response;
            }
            
            // 服务器错误（5xx）或其他错误，继续重试
            console.warn(`[fetchWithRetry] 请求失败 ${response.status}, 准备重试...`);
            
        } catch (error) {
            console.warn(`[fetchWithRetry] 第${i + 1}次请求异常:`, error.message);
            
            // 如果是最后一次重试，抛出错误
            if (i === maxRetries - 1) {
                throw error;
            }
        }
        
        // 指数退避：等待 1s, 2s, 4s, 8s...，最多10s
        const delay = Math.min(1000 * Math.pow(2, i), 10000);
        console.log(`[fetchWithRetry] 等待 ${delay}ms 后重试...`);
        await new Promise(resolve => setTimeout(resolve, delay));
    }
    
    throw new Error(`请求失败，已重试${maxRetries}次`);
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    window.releaseManager = new ReleaseListManager();
    window.releaseCreateManager = new ReleaseCreateManager(window.releaseManager);
    window.nodeSimulator = new NodeSimulator();
    
    // Tab切换逻辑
    const tabReleaseList = document.getElementById('tabReleaseList');
    const tabNodeSimulator = document.getElementById('tabNodeSimulator');
    const releaseListSection = document.getElementById('releaseListSection');
    const nodeSimulatorSection = document.getElementById('nodeSimulatorSection');
    
    function showReleaseList() {
        tabReleaseList.classList.add('active');
        tabNodeSimulator.classList.remove('active');
        releaseListSection.classList.remove('hidden');
        nodeSimulatorSection.classList.add('hidden');
    }
    
    function showNodeSimulator() {
        tabNodeSimulator.classList.add('active');
        tabReleaseList.classList.remove('active');
        releaseListSection.classList.add('hidden');
        nodeSimulatorSection.classList.remove('hidden');
    }
    
    tabReleaseList.addEventListener('click', showReleaseList);
    tabNodeSimulator.addEventListener('click', showNodeSimulator);
    
    // 检查URL hash，如果是#simulator，则自动切换到节点模拟器Tab
    if (window.location.hash === '#simulator') {
        showNodeSimulator();
        // 清除hash，避免下次刷新时再次进入
        history.replaceState(null, null, ' ');
    }
});

// ========== 节点模拟器类 ==========
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
        const startBtn = document.getElementById('startSimulationBtn');
        if (startBtn) {
            startBtn.addEventListener('click', () => {
                this.startSimulation();
            });
        }

        // 随机选择节点
        const randomBtn = document.getElementById('randomSelectBtn');
        if (randomBtn) {
            randomBtn.addEventListener('click', () => {
                this.randomSelectNodes();
            });
        }

        // 停止模拟
        const stopBtn = document.getElementById('stopSimulationBtn');
        if (stopBtn) {
            stopBtn.addEventListener('click', () => {
                this.stopSimulation();
            });
        }
    }

    // 添加节点日志
    addNodeLog(nodeId, message, type = 'info') {
        const node = this.nodes.get(nodeId);
        if (!node) return;

        const timestamp = new Date().toLocaleTimeString('zh-CN', { hour12: false });
        const logEntry = {
            timestamp: timestamp,
            message: message,
            type: type
        };
        
        // 保存到节点数据中
        node.logs.push(logEntry);
        
        // 更新DOM中的日志显示
        const logsContainer = document.getElementById(`node-logs-${nodeId}`);
        if (logsContainer) {
            const logDiv = document.createElement('div');
            logDiv.className = `log-entry log-${type}`;
            logDiv.innerHTML = `<span class="log-timestamp">[${timestamp}]</span>${message}`;
            logsContainer.appendChild(logDiv);
            logsContainer.scrollTop = logsContainer.scrollHeight;
        }
    }

    // 清空节点日志
    clearNodeLogs(nodeId) {
        const node = this.nodes.get(nodeId);
        if (node) {
            node.logs = [];
        }
        
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

        this.stopSimulation();

        this.nodes.clear();
        nodeIds.forEach(nodeId => {
            this.nodes.set(nodeId, {
                nodeId: nodeId,
                status: 'active',
                components: {},
                lastUpdate: null,
                logs: []  // 保存日志历史
            });
        });

        this.renderNodes();
        this.updateStatus(true);

        for (const nodeId of nodeIds) {
            await this.fetchNodeConfig(nodeId);
        }

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
            const components = this.parseComponents(config);
            
            const node = this.nodes.get(nodeId);
            if (node) {
                const oldComponents = node.components;

                // 检查是否是首次获取配置
                if (Object.keys(oldComponents).length === 0) {
                    this.addNodeLog(nodeId, `配置已更新, 共 ${Object.keys(components).length} 个组件`, 'info');
                } else {
                    // 检测版本变化 - 遍历所有新配置中的组件
                    Object.keys(components).forEach(appName => {
                        const oldVersion = oldComponents[appName];
                        const newVersion = components[appName];
                        
                        // 只有当旧版本存在且与新版本不同时才输出日志
                        if (oldVersion && oldVersion !== newVersion) {
                            this.addNodeLog(nodeId, `检测到版本更新, 组件${appName}升级到版本${newVersion} ( ${oldVersion} -> ${newVersion} )`, 'success');
                        }
                    });
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
            if (app) {
                // 优先使用alter的url，如果不存在则使用main的url
                let url = null;
                if (app.alter && app.alter.url) {
                    url = app.alter.url;
                } else if (app.main && app.main.url) {
                    url = app.main.url;
                }
                
                if (url) {
                    const version = this.extractVersionFromUrl(url);
                    components[appName] = version;
                }
            }
        });

        return components;
    }

    // 从URL中提取版本号
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
                            <button class="btn btn-sm btn-secondary" onclick="window.nodeSimulator.clearNodeLogs('${nodeId}')">清空</button>
                        </div>
                        <div class="node-logs-container" id="node-logs-${nodeId}"></div>
                    </div>
                </div>
            `;

            container.appendChild(nodeCard);
            
            // 恢复日志
            const logsContainer = document.getElementById(`node-logs-${nodeId}`);
            if (logsContainer && node.logs) {
                node.logs.forEach(log => {
                    const logDiv = document.createElement('div');
                    logDiv.className = `log-entry log-${log.type}`;
                    logDiv.innerHTML = `<span class="log-timestamp">[${log.timestamp}]</span>${log.message}`;
                    logsContainer.appendChild(logDiv);
                });
                logsContainer.scrollTop = logsContainer.scrollHeight;
            }
        });

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

// 导出到全局作用域，供HTML中的onclick使用
window.releaseManager = null;
window.releaseCreateManager = null;
window.nodeSimulator = null;