class ComponentManagement {
    constructor() {
        this.currentPage = 1;
        this.pageSize = 10;
        this.totalCount = 0;
        this.totalPages = 0;
        this.currentFilters = {
            app: '',
            nodeType: ''
        };
        this.editingComponent = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadData();
    }

    bindEvents() {
        // 搜索按钮
        document.getElementById('searchBtn').addEventListener('click', () => {
            this.handleSearch();
        });

        // 回车搜索
        document.getElementById('appName').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.handleSearch();
            }
        });

        // 新增组件按钮
        document.getElementById('addComponentBtn').addEventListener('click', () => {
            this.showAddModal();
        });

        // 分页按钮
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

        // 弹窗相关事件
        this.bindModalEvents();
    }

    bindModalEvents() {
        // 关闭弹窗
        document.getElementById('closeModal').addEventListener('click', () => {
            this.hideModal();
        });

        document.getElementById('cancelBtn').addEventListener('click', () => {
            this.hideModal();
        });

        // 保存按钮
        document.getElementById('saveBtn').addEventListener('click', () => {
            this.handleSave();
        });

        // 删除弹窗
        document.getElementById('closeDeleteModal').addEventListener('click', () => {
            this.hideDeleteModal();
        });

        document.getElementById('cancelDeleteBtn').addEventListener('click', () => {
            this.hideDeleteModal();
        });

        document.getElementById('confirmDeleteBtn').addEventListener('click', () => {
            this.handleDelete();
        });

        // 点击弹窗外部关闭
        document.getElementById('componentModal').addEventListener('click', (e) => {
            if (e.target.id === 'componentModal') {
                this.hideModal();
            }
        });

        document.getElementById('deleteModal').addEventListener('click', (e) => {
            if (e.target.id === 'deleteModal') {
                this.hideDeleteModal();
            }
        });

        // 初始化多选下拉框
        this.initMultiSelect();
    }

    async loadData() {
        try {
            this.showLoading();
            
            const params = new URLSearchParams({
                ...this.currentFilters
            });

            // 如果有筛选条件，添加到请求参数中
            if (this.currentFilters.app) {
                params.append('app', this.currentFilters.app);
            }
            if (this.currentFilters.nodeType) {
                params.append('nodeType', this.currentFilters.nodeType);
            }

            const url = `/api/v1/release/allowapps?${params.toString()}`;
            const response = await fetch(url, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            
            // 检查响应格式
            if (data.code && data.code !== 0) {
                throw new Error(data.desc || '请求失败');
            }

            // 处理数据
            const apps = data.apps || [];
            this.totalCount = apps.length;
            
            // 前端分页处理
            this.totalPages = Math.ceil(this.totalCount / this.pageSize);
            const startIndex = (this.currentPage - 1) * this.pageSize;
            const endIndex = startIndex + this.pageSize;
            const pageData = apps.slice(startIndex, endIndex);

            this.renderTable(pageData);
            this.renderPagination();

        } catch (error) {
            console.error('加载数据失败:', error);
            this.showError('加载数据失败: ' + error.message);
            this.renderTable([]);
        } finally {
            this.hideLoading();
        }
    }

    // 初始化多选下拉框
    initMultiSelect() {
        const display = document.getElementById('nodeTypeDisplay');
        const dropdown = document.getElementById('nodeTypeDropdown');
        
        if (!display || !dropdown) return;
        
        const options = dropdown.querySelectorAll('.multi-select-option');

        // 点击显示区域切换下拉框
        display.addEventListener('click', (e) => {
            e.stopPropagation();
            this.toggleDropdown();
        });

        // 点击选项
        options.forEach(option => {
            option.addEventListener('click', (e) => {
                e.stopPropagation();
                const checkbox = option.querySelector('input[type="checkbox"]');
                checkbox.checked = !checkbox.checked;
                this.updateSelectedDisplay();
            });
        });

        // 点击文档其他地方关闭下拉框
        document.addEventListener('click', () => {
            this.closeDropdown();
        });
    }

    // 切换下拉框显示状态
    toggleDropdown() {
        const display = document.getElementById('nodeTypeDisplay');
        const dropdown = document.getElementById('nodeTypeDropdown');
        
        if (dropdown.classList.contains('show')) {
            this.closeDropdown();
        } else {
            display.classList.add('active');
            dropdown.classList.add('show');
        }
    }

    // 关闭下拉框
    closeDropdown() {
        const display = document.getElementById('nodeTypeDisplay');
        const dropdown = document.getElementById('nodeTypeDropdown');
        
        display.classList.remove('active');
        dropdown.classList.remove('show');
    }

    // 更新选中项显示
    updateSelectedDisplay() {
        const display = document.getElementById('nodeTypeDisplay');
        const dropdown = document.getElementById('nodeTypeDropdown');
        const checkboxes = dropdown.querySelectorAll('input[type="checkbox"]:checked');
        
        // 清空当前显示
        const existingItems = display.querySelector('.selected-items');
        const placeholder = display.querySelector('.placeholder');
        
        if (existingItems) {
            existingItems.remove();
        }
        
        if (checkboxes.length > 0) {
            // 隐藏占位符
            if (placeholder) {
                placeholder.style.display = 'none';
            }
            
            // 创建选中项容器
            const selectedContainer = document.createElement('div');
            selectedContainer.className = 'selected-items';
            
            checkboxes.forEach(checkbox => {
                const option = checkbox.closest('.multi-select-option');
                const label = option.querySelector('label').textContent;
                
                const tag = document.createElement('span');
                tag.className = 'selected-tag';
                tag.innerHTML = `
                    ${label}
                    <span class="remove" data-value="${checkbox.value}">×</span>
                `;
                
                // 添加移除功能
                tag.querySelector('.remove').addEventListener('click', (e) => {
                    e.stopPropagation();
                    checkbox.checked = false;
                    this.updateSelectedDisplay();
                });
                
                selectedContainer.appendChild(tag);
            });
            
            display.insertBefore(selectedContainer, display.querySelector('.arrow'));
        } else {
            // 显示占位符
            if (placeholder) {
                placeholder.style.display = 'block';
            }
        }
    }

    renderTable(data) {
        const tbody = document.getElementById('tableBody');
        
        if (!data || data.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="7" class="empty-state">
                        <div class="empty-state-icon">📦</div>
                        <div class="empty-state-text">暂无组件数据</div>
                    </td>
                </tr>
            `;
            return;
        }

        // 存储组件数据以供编辑使用
        this.componentsData = data;

        tbody.innerHTML = data.map((item, index) => `
            <tr>
                <td>${this.escapeHtml(item.name || '')}</td>
                <td>${this.formatDeviceType(item.nodeType || '')}</td>
                <td>${this.escapeHtml(item.path || '')}</td>
                <td>${this.escapeHtml(item.desc || '')}</td>
                <td>${this.escapeHtml(item.operator || '')}</td>
                <td>${this.formatDate(item.createAt)}</td>
                <td>
                    <button class="action-btn edit-btn" onclick="componentManager.showEditModal(${index})">
                        编辑
                    </button>
                    <button class="action-btn delete-btn" onclick="componentManager.showDeleteModal('${item.id}', '${this.escapeHtml(item.name)}')">
                        删除
                    </button>
                </td>
            </tr>
        `).join('');
    }

    renderPagination() {
        // 更新总数显示
        document.getElementById('totalCount').textContent = this.totalCount;
        document.getElementById('totalPages').textContent = this.totalPages;

        // 更新分页按钮状态
        const prevBtn = document.getElementById('prevPage');
        const nextBtn = document.getElementById('nextPage');
        
        prevBtn.disabled = this.currentPage <= 1;
        nextBtn.disabled = this.currentPage >= this.totalPages;

        // 生成页码
        this.renderPageNumbers();
    }

    renderPageNumbers() {
        const pageNumbers = document.getElementById('pageNumbers');
        const maxVisiblePages = 5;
        let startPage = Math.max(1, this.currentPage - Math.floor(maxVisiblePages / 2));
        let endPage = Math.min(this.totalPages, startPage + maxVisiblePages - 1);

        if (endPage - startPage + 1 < maxVisiblePages) {
            startPage = Math.max(1, endPage - maxVisiblePages + 1);
        }

        let html = '';
        
        for (let i = startPage; i <= endPage; i++) {
            html += `
                <button class="page-number ${i === this.currentPage ? 'active' : ''}" 
                        onclick="componentManager.goToPage(${i})">
                    ${i}
                </button>
            `;
        }

        pageNumbers.innerHTML = html;
    }

    goToPage(page) {
        if (page >= 1 && page <= this.totalPages && page !== this.currentPage) {
            this.currentPage = page;
            this.loadData();
        }
    }

    handleSearch() {
        this.currentFilters.app = document.getElementById('appName').value.trim();
        this.currentFilters.nodeType = document.getElementById('nodeType').value;
        this.currentPage = 1; // 重置到第一页
        this.loadData();
    }

    showAddModal() {
        this.editingComponent = null;
        const modalTitle = document.querySelector('#componentModal .modal-title');
        if (modalTitle) {
            modalTitle.textContent = '新增组件';
        }
        this.resetForm();
        this.showModal();
    }

    async showEditModal(componentIndex) {
        try {
            // 直接使用列表数据，不请求详情接口
            if (!this.componentsData || !this.componentsData[componentIndex]) {
                throw new Error('组件数据不存在');
            }

            const component = this.componentsData[componentIndex];
            
            // 设置编辑模式
            this.editingComponent = { id: component.id };
            
            // 设置模态框标题
            const modalTitle = document.querySelector('#componentModal .modal-title');
            if (modalTitle) {
                modalTitle.textContent = '编辑组件';
            }
            
            // 填充表单数据
            this.fillFormData(component);
            
            // 显示模态框
            this.showModal();
            
        } catch (error) {
            console.error('编辑组件失败:', error);
            this.showError('编辑组件失败: ' + error.message);
        }
    }

    fillFormData(component) {
        // 填充基本信息
        document.getElementById('componentName').value = component.name || '';
        document.getElementById('componentPath').value = component.path || '';
        document.getElementById('componentDesc').value = component.desc || '';
        
        // 在编辑模式下，组件名和设备类型不可修改
        const nameInput = document.getElementById('componentName');
        const nodeTypeContainer = document.querySelector('.multi-select-container');
        
        if (this.editingComponent && this.editingComponent.id) {
            // 编辑模式：禁用组件名和设备类型
            nameInput.disabled = true;
            nameInput.style.backgroundColor = '#f5f5f5';
            nameInput.style.cursor = 'not-allowed';
            
            if (nodeTypeContainer) {
                nodeTypeContainer.style.pointerEvents = 'none';
                nodeTypeContainer.style.opacity = '0.6';
                nodeTypeContainer.style.cursor = 'not-allowed';
            }
        } else {
            // 新增模式：启用所有字段
            nameInput.disabled = false;
            nameInput.style.backgroundColor = '';
            nameInput.style.cursor = '';
            
            if (nodeTypeContainer) {
                nodeTypeContainer.style.pointerEvents = '';
                nodeTypeContainer.style.opacity = '';
                nodeTypeContainer.style.cursor = '';
            }
        }
        
        // 填充设备类型
        this.setSelectedNodeTypes(component.nodeType);
    }

    setSelectedNodeTypes(nodeType) {
        // 清除所有选中状态
        const checkboxes = document.querySelectorAll('#nodeTypeDropdown input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = false;
        });
        
        // 设置当前组件的设备类型为选中状态
        if (nodeType) {
            const checkbox = document.querySelector(`#nodeTypeDropdown input[value="${nodeType}"]`);
            if (checkbox) {
                checkbox.checked = true;
            }
        }
        
        // 更新显示
        this.updateSelectedDisplay();
    }

    showDeleteModal(componentId, componentName) {
        this.editingComponent = { id: componentId, name: componentName };
        document.getElementById('deleteComponentName').textContent = componentName;
        document.getElementById('deleteModal').classList.remove('hidden');
    }

    showModal() {
        document.getElementById('componentModal').classList.remove('hidden');
    }

    hideModal() {
        document.getElementById('componentModal').classList.add('hidden');
        this.resetForm();
    }

    hideDeleteModal() {
        document.getElementById('deleteModal').classList.add('hidden');
        this.editingComponent = null;
    }

    resetForm() {
        const form = document.getElementById('componentForm');
        if (form) {
            form.reset();
        }
        
        // 清除多选下拉框的选中状态
        const dropdown = document.getElementById('nodeTypeDropdown');
        if (dropdown) {
            const checkboxes = dropdown.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach(cb => cb.checked = false);
        }
        this.updateSelectedDisplay();
        
        // 重置字段状态（确保新增模式下字段可编辑）
        const nameInput = document.getElementById('componentName');
        const nodeTypeContainer = document.querySelector('.multi-select-container');
        
        if (nameInput) {
            nameInput.disabled = false;
            nameInput.style.backgroundColor = '';
            nameInput.style.cursor = '';
        }
        
        if (nodeTypeContainer) {
            nodeTypeContainer.style.pointerEvents = '';
            nodeTypeContainer.style.opacity = '';
            nodeTypeContainer.style.cursor = '';
        }
        
        this.currentEditId = null;
    }

    async handleSave() {
        try {
            const formData = this.getFormData();
            
            if (!this.validateForm(formData)) {
                return;
            }

            this.showLoading();

            const isEdit = this.editingComponent && this.editingComponent.id;
            const method = 'POST'; // 根据API文档，新增和编辑都使用POST
            const url = '/api/v1/release/allowapps';

            const requestData = {
                operation: isEdit ? 'update' : 'add',
                id: isEdit ? this.editingComponent.id : undefined,
                name: formData.name,
                nodeTypes: formData.nodeTypes,
                path: formData.path,
                desc: formData.desc
            };

            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            // 检查响应是否有内容
            const responseText = await response.text();
            let data = {};
            
            if (responseText.trim()) {
                try {
                    data = JSON.parse(responseText);
                } catch (e) {
                    console.warn('Failed to parse JSON response:', responseText);
                    // 如果解析失败但状态码是200，认为操作成功
                    data = {};
                }
            }
            
            if (data.code && data.code !== 0) {
                throw new Error(data.desc || '操作失败');
            }

            this.hideModal();
            this.loadData(); // 重新加载数据
            this.showSuccess(isEdit ? '编辑成功' : '新增成功');

        } catch (error) {
            console.error('保存失败:', error);
            this.showError('保存失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    async handleDelete() {
        try {
            if (!this.editingComponent || !this.editingComponent.id) {
                return;
            }

            this.showLoading();

            const requestData = {
                operation: 'del',  // 修改为 'del' 而不是 'delete'
                id: this.editingComponent.id
            };

            const response = await fetch('/api/v1/release/allowapps', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            // 检查响应是否有内容 (与保存操作一样处理)
            const responseText = await response.text();
            let data = {};
            
            if (responseText.trim()) {
                try {
                    data = JSON.parse(responseText);
                } catch (e) {
                    console.warn('Failed to parse JSON response:', responseText);
                    // 如果解析失败但状态码是200，认为操作成功
                    data = {};
                }
            }
            
            if (data.code && data.code !== 0) {
                throw new Error(data.desc || '删除失败');
            }

            this.hideDeleteModal();
            this.loadData(); // 重新加载数据
            this.showSuccess('删除成功');

        } catch (error) {
            console.error('删除失败:', error);
            this.showError('删除失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    getFormData() {
        const form = document.getElementById('componentForm');
        const formData = new FormData(form);
        
        // 获取选中的设备类型（从多选下拉框）
        const nodeTypes = [];
        const dropdown = document.getElementById('nodeTypeDropdown');
        if (dropdown) {
            const nodeTypeCheckboxes = dropdown.querySelectorAll('input[type="checkbox"]:checked');
            nodeTypeCheckboxes.forEach(cb => nodeTypes.push(cb.value));
        }

        // 对于编辑模式，需要手动获取禁用字段的值
        let name = formData.get('name');
        if (!name && this.editingComponent && this.editingComponent.id) {
            // 编辑模式下，组件名字段被禁用，需要手动获取值
            const nameInput = document.getElementById('componentName');
            name = nameInput ? nameInput.value : '';
        }

        return {
            name: name,
            nodeTypes: nodeTypes,
            path: formData.get('path'),
            desc: formData.get('desc')
        };
    }

    validateForm(formData) {
        if (!formData.name || !formData.name.trim()) {
            this.showError('请输入组件名');
            return false;
        }

        if (!formData.nodeTypes || formData.nodeTypes.length === 0) {
            this.showError('请选择至少一个设备类型');
            return false;
        }

        if (!formData.path || !formData.path.trim()) {
            this.showError('请输入存放路径');
            return false;
        }

        return true;
    }

    showLoading() {
        document.getElementById('loading').classList.remove('hidden');
    }

    hideLoading() {
        document.getElementById('loading').classList.add('hidden');
    }

    showError(message) {
        // 简单的错误提示，可以后续优化为更好的UI组件
        alert('错误: ' + message);
    }

    showSuccess(message) {
        // 简单的成功提示，可以后续优化为更好的UI组件
        alert('成功: ' + message);
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(timestamp) {
        if (!timestamp) return '';
        
        try {
            // 后端返回的是Unix时间戳（秒），需要转换为毫秒
            const date = new Date(timestamp * 1000);
            
            // 格式化为 YYYY-MM-DD HH:mm:ss
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const hours = String(date.getHours()).padStart(2, '0');
            const minutes = String(date.getMinutes()).padStart(2, '0');
            const seconds = String(date.getSeconds()).padStart(2, '0');
            
            return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
        } catch (error) {
            return timestamp;
        }
    }

    formatDeviceType(nodeType) {
        const deviceTypeMap = {
            'node': '大节点',
            'smallBox': '小盒子'
        };
        return deviceTypeMap[nodeType] || nodeType;
    }
}

// 初始化组件管理器
let componentManager;

document.addEventListener('DOMContentLoaded', () => {
    componentManager = new ComponentManagement();
});