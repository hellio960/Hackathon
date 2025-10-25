class ReleaseDetailManager {
    constructor() {
        this.releaseId = null;
        this.detailData = null;
        this.confirmCallback = null;
        
        this.init();
    }

    init() {
        // 从URL获取releaseId
        this.releaseId = this.getReleaseIdFromUrl();
        
        if (!this.releaseId) {
            this.showError('缺少发布任务ID');
            setTimeout(() => {
                window.location.href = 'release-list.html';
            }, 2000);
            return;
        }

        this.bindEvents();
        this.loadDetail();
    }

    // 从URL获取releaseId
    getReleaseIdFromUrl() {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('id');
    }

    bindEvents() {
        // 刷新按钮
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.loadDetail();
        });

        // 切换灰度模式按钮
        document.getElementById('switchModeBtn').addEventListener('click', () => {
            this.showSwitchModeModal();
        });

        // 导出节点按钮
        document.getElementById('exportSpecifiedNodesBtn').addEventListener('click', () => {
            this.exportNodes();
        });

        document.getElementById('exportCurrentNodesBtn').addEventListener('click', () => {
            this.exportNodes();
        });

        // 操作历史按钮
        document.getElementById('showHistoryBtn').addEventListener('click', () => {
            this.showHistoryModal();
        });

        // 主要操作按钮（验证完成/全量发布/回滚）
        document.getElementById('primaryActionBtn').addEventListener('click', () => {
            this.handlePrimaryAction();
        });

        // 继续灰度按钮
        document.getElementById('continueBtn').addEventListener('click', () => {
            this.handleContinue();
        });

        // 切换模式弹窗
        document.getElementById('closeSwitchModeModal').addEventListener('click', () => {
            this.hideSwitchModeModal();
        });

        document.getElementById('cancelSwitchModeBtn').addEventListener('click', () => {
            this.hideSwitchModeModal();
        });

        document.getElementById('confirmSwitchModeBtn').addEventListener('click', () => {
            this.handleSwitchMode();
        });

        // 切换模式单选按钮
        document.querySelectorAll('input[name="switchGrayMode"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.toggleSwitchMode(e.target.value);
            });
        });

        // 历史弹窗
        document.getElementById('closeHistoryModal').addEventListener('click', () => {
            this.hideHistoryModal();
        });

        document.getElementById('closeHistoryBtn').addEventListener('click', () => {
            this.hideHistoryModal();
        });

        // 确认弹窗
        document.getElementById('closeConfirmModal').addEventListener('click', () => {
            this.hideConfirmModal();
        });

        document.getElementById('cancelConfirmBtn').addEventListener('click', () => {
            this.hideConfirmModal();
        });

        document.getElementById('doConfirmBtn').addEventListener('click', () => {
            if (this.confirmCallback) {
                this.confirmCallback();
            }
            this.hideConfirmModal();
        });

        // 点击弹窗外部关闭
        document.getElementById('switchModeModal').addEventListener('click', (e) => {
            if (e.target.id === 'switchModeModal') {
                this.hideSwitchModeModal();
            }
        });

        document.getElementById('historyModal').addEventListener('click', (e) => {
            if (e.target.id === 'historyModal') {
                this.hideHistoryModal();
            }
        });

        document.getElementById('confirmModal').addEventListener('click', (e) => {
            if (e.target.id === 'confirmModal') {
                this.hideConfirmModal();
            }
        });

        // 初始化节点阶段和业务ID自定义多选下拉框
        this.initCustomMultiSelect('switchStagesWrapper');
        this.initCustomMultiSelect('switchCustomerIdsWrapper');
        
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

    // 加载发布详情
    async loadDetail() {
        try {
            this.showLoading();

            const response = await fetch(`/api/v1/release/${this.releaseId}/detail`, {
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
                throw new Error(data.desc || '获取发布详情失败');
            }

            this.detailData = data;
            this.renderDetail(data);

        } catch (error) {
            console.error('加载发布详情失败:', error);
            this.showError('加载发布详情失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 渲染详情
    renderDetail(data) {
        // 基础信息
        document.getElementById('releaseId').textContent = data.id || '-';
        document.getElementById('appName').textContent = data.app || '-';
        document.getElementById('deviceType').textContent = this.getDeviceTypeText(data.deviceType) || '-';
        document.getElementById('opType').textContent = this.getOpTypeText(data.opType) || '-';
        document.getElementById('updateTime').textContent = this.formatDateTime(data.updateAt) || '-';
        document.getElementById('operator').textContent = data.operator || '-';
        document.getElementById('releaseType').textContent = this.getReleaseTypeText(data.releaseType) || '-';
        document.getElementById('releaseState').textContent = this.getStateText(data.state) || '-';
        document.getElementById('createTime').textContent = this.formatDateTime(data.createAt) || '-';

        // 当前运行版本
        if (data.mainConfig) {
            document.getElementById('mainUrl').textContent = data.mainConfig.url || '-';
            document.getElementById('mainMd5').textContent = data.mainConfig.md5 || '-';
            document.getElementById('mainType').textContent = this.getPackageTypeText(data.mainConfig.type) || '-';
            document.getElementById('mainCmd').textContent = data.mainConfig.cmd || '-';
            document.getElementById('mainArgs').textContent = (data.mainConfig.args || []).join(' ') || '-';
            document.getElementById('mainWorkDir').textContent = data.mainConfig.dir || '-';
        } else {
            this.clearConfig('main');
        }

        // 灰度版本
        if (data.alterConfig) {
            document.getElementById('alterUrl').textContent = data.alterConfig.url || '-';
            document.getElementById('alterMd5').textContent = data.alterConfig.md5 || '-';
            document.getElementById('alterType').textContent = this.getPackageTypeText(data.alterConfig.type) || '-';
            document.getElementById('alterCmd').textContent = data.alterConfig.cmd || '-';
            document.getElementById('alterArgs').textContent = (data.alterConfig.args || []).join(' ') || '-';
            document.getElementById('alterWorkDir').textContent = data.alterConfig.dir || '-';
        } else {
            this.clearConfig('alter');
        }

        // 灰度策略
        this.renderGrayPolicy(data);

        // 设置操作按钮
        this.setupActionButtons(data);

        // 设置灰度比例默认值
        document.getElementById('newPercentage').value = data.grayPolicy?.percentage || 0;
    }

    // 清空配置显示
    clearConfig(type) {
        const prefix = type === 'main' ? 'main' : 'alter';
        document.getElementById(`${prefix}Url`).textContent = '-';
        document.getElementById(`${prefix}Md5`).textContent = '-';
        document.getElementById(`${prefix}Type`).textContent = '-';
        document.getElementById(`${prefix}Cmd`).textContent = '-';
        document.getElementById(`${prefix}Args`).textContent = '-';
        document.getElementById(`${prefix}WorkDir`).textContent = '-';
    }

    // 渲染灰度策略
    renderGrayPolicy(data) {
        const policy = data.grayPolicy;
        
        if (!policy) {
            document.getElementById('grayMode').textContent = '-';
            this.hideFilterRows();
            
            const specifiedNodesEl = document.getElementById('specifiedNodesList');
            specifiedNodesEl.textContent = '-';
            specifiedNodesEl.classList.add('empty');
            
            const currentNodesEl = document.getElementById('currentNodesList');
            currentNodesEl.textContent = '-';
            currentNodesEl.classList.add('empty');
            
            document.getElementById('currentPercentage').textContent = '0';
            document.getElementById('allowCount').textContent = '0';
            document.getElementById('totalCount').textContent = '0';
            return;
        }

        // 灰度方式
        const hasNodeIds = policy.nodeIds && policy.nodeIds.length > 0;
        const hasFilter = policy.filter && Object.keys(policy.filter).length > 0;
        
        if (hasNodeIds && !hasFilter) {
            document.getElementById('grayMode').textContent = '灰度节点';
            this.hideFilterRows();
        } else if (!hasNodeIds && hasFilter) {
            document.getElementById('grayMode').textContent = '规则过滤';
            this.showFilterRows(policy.filter);
        } else if (hasNodeIds && hasFilter) {
            document.getElementById('grayMode').textContent = '混合模式';
            this.showFilterRows(policy.filter);
        } else {
            document.getElementById('grayMode').textContent = '-';
            this.hideFilterRows();
        }

        // 指定灰度节点
        const specifiedNodesEl = document.getElementById('specifiedNodesList');
        if (policy.nodeIds && policy.nodeIds.length > 0) {
            specifiedNodesEl.textContent = policy.nodeIds.join('\n');
            specifiedNodesEl.classList.remove('empty');
            document.getElementById('specifiedNodesRow').style.display = '';
        } else {
            specifiedNodesEl.textContent = '-';
            specifiedNodesEl.classList.add('empty');
            // 如果是规则过滤且没有指定节点，隐藏该行
            if (!hasNodeIds && hasFilter) {
                document.getElementById('specifiedNodesRow').style.display = 'none';
            } else {
                document.getElementById('specifiedNodesRow').style.display = '';
            }
        }

        // 当前灰度节点（显示前100个）
        const currentNodesEl = document.getElementById('currentNodesList');
        if (data.allowNodes && data.allowNodes.length > 0) {
            const displayNodes = data.allowNodes.slice(0, 100);
            let text = displayNodes.join('\n');
            if (data.allowNodesTotal > 100) {
                text += `\n\n... (共 ${data.allowNodesTotal} 个节点，仅显示前100个)`;
            }
            currentNodesEl.textContent = text;
            currentNodesEl.classList.remove('empty');
        } else {
            currentNodesEl.textContent = '-';
            currentNodesEl.classList.add('empty');
        }

        // 统计信息
        document.getElementById('currentPercentage').textContent = policy.percentage || 0;
        document.getElementById('allowCount').textContent = data.allowNodesTotal || 0;
        document.getElementById('totalCount').textContent = policy.statInfo?.totalCount || 0;
    }

    // 设置操作按钮
    setupActionButtons(data) {
        const primaryBtn = document.getElementById('primaryActionBtn');
        const continueBtn = document.getElementById('continueBtn');
        
        const releaseType = data.releaseType;
        const state = data.state;
        const rollbackAllowed = data.rollbackAllowed;

        // 根据发布类型和状态设置主要操作按钮
        if (releaseType === 'beta') {
            // 功能验证：显示"验证完成"按钮
            primaryBtn.textContent = '验证完成';
            primaryBtn.className = 'btn btn-danger';
            primaryBtn.style.display = 'inline-block';
        } else if (releaseType === 'formal') {
            // 正式发布
            if (state === 'processing') {
                // 进行中：显示"全量发布"按钮
                primaryBtn.textContent = '全量发布';
                primaryBtn.className = 'btn btn-danger';
                primaryBtn.style.display = 'inline-block';
            } else if (state === 'complete' && rollbackAllowed) {
                // 已完成且允许回滚：显示"回滚"按钮
                primaryBtn.textContent = '回滚';
                primaryBtn.className = 'btn btn-danger';
                primaryBtn.style.display = 'inline-block';
            } else {
                // 其他状态：隐藏主要操作按钮
                primaryBtn.style.display = 'none';
            }
        }

        // 继续灰度按钮：只在进行中状态显示
        if (state === 'processing') {
            continueBtn.style.display = 'inline-block';
        } else {
            continueBtn.style.display = 'none';
        }
    }

    // 显示过滤规则行
    showFilterRows(filter) {
        if (!filter) {
            this.hideFilterRows();
            return;
        }

        // 设备类型
        if (filter.devType) {
            const devTypeMap = {
                'ant.A': '安卓 arm32',
                'ant.B': '安卓 arm64',
                'ant.C': '安卓 amd64',
                'jarvis.A': '大节点 x86'
            };
            document.getElementById('filterDevType').textContent = devTypeMap[filter.devType] || filter.devType;
            document.getElementById('filterDevTypeRow').style.display = '';
        } else {
            document.getElementById('filterDevTypeRow').style.display = 'none';
        }

        // 节点状态
        if (filter.status) {
            const statusMap = {
                'online': '在线',
                'outline': '离线'
            };
            document.getElementById('filterStatus').textContent = statusMap[filter.status] || filter.status;
            document.getElementById('filterStatusRow').style.display = '';
        } else {
            document.getElementById('filterStatusRow').style.display = 'none';
        }

        // 节点阶段
        if (filter.stages && filter.stages.length > 0) {
            const stageMap = {
                'register': '已注册',
                'submitted': '已提交待验收',
                'censored': '验收通过',
                'uncensored': '验收不通过',
                'inservice': '服务中'
            };
            const stageLabels = filter.stages.map(s => stageMap[s] || s);
            document.getElementById('filterStages').textContent = stageLabels.join(', ');
            document.getElementById('filterStagesRow').style.display = '';
        } else {
            document.getElementById('filterStagesRow').style.display = 'none';
        }

        // 业务ID
        if (filter.customerIds && filter.customerIds.length > 0) {
            document.getElementById('filterCustomerIds').textContent = filter.customerIds.join(', ');
            document.getElementById('filterCustomerIdsRow').style.display = '';
        } else {
            document.getElementById('filterCustomerIdsRow').style.display = 'none';
        }
    }

    // 隐藏所有过滤规则行
    hideFilterRows() {
        document.getElementById('filterDevTypeRow').style.display = 'none';
        document.getElementById('filterStatusRow').style.display = 'none';
        document.getElementById('filterStagesRow').style.display = 'none';
        document.getElementById('filterCustomerIdsRow').style.display = 'none';
    }

    // 处理主要操作（验证完成/全量发布/回滚）
    handlePrimaryAction() {
        const releaseType = this.detailData.releaseType;
        const state = this.detailData.state;
        
        if (releaseType === 'beta') {
            // 功能验证：验证完成（调用complete接口）
            this.showConfirm(
                '确认验证完成',
                '确定要标记此功能验证任务为已完成吗？',
                () => this.doComplete()
            );
        } else if (releaseType === 'formal') {
            if (state === 'processing') {
                // 正式发布进行中：全量发布
                this.showConfirm(
                    '确认全量发布',
                    '确定要进行全量发布吗？此操作将把新版本推送到所有节点。',
                    () => this.doComplete()
                );
            } else if (state === 'complete') {
                // 正式发布已完成：回滚
                this.showConfirm(
                    '确认回滚',
                    '确定要回滚到上一个版本吗？此操作不可撤销。',
                    () => this.doRollback()
                );
            }
        }
    }

    // 继续灰度
    async handleContinue() {
        const percentage = parseInt(document.getElementById('newPercentage').value);
        const addNodesText = document.getElementById('addNodes').value.trim();
        const removeNodesText = document.getElementById('removeNodes').value.trim();

        if (isNaN(percentage) || percentage < 0 || percentage > 100) {
            this.showError('请输入有效的灰度比例（0-100）');
            return;
        }

        const addNodes = addNodesText ? addNodesText.split('\n').map(n => n.trim()).filter(n => n) : [];
        const delNodes = removeNodesText ? removeNodesText.split('\n').map(n => n.trim()).filter(n => n) : [];

        try {
            this.showLoading();

            const requestData = {
                percentage: percentage,
                addNodes: addNodes,
                delNodes: delNodes
            };

            console.log('继续灰度请求:', requestData);

            const response = await fetch(`/api/v1/release/${this.releaseId}/continue`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            console.log('继续灰度响应状态:', response.status, response.statusText);

            const responseText = await response.text();
            console.log('继续灰度响应内容:', responseText || '(空响应)');

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

            // 成功响应：可能返回空响应
            if (responseText && responseText.trim() !== '') {
                try {
                    const data = JSON.parse(responseText);
                    if (data.code && data.code !== 0) {
                        throw new Error(data.desc || '继续灰度失败');
                    }
                } catch (parseError) {
                    console.warn('解析响应数据失败，但请求已成功:', parseError);
                }
            }

            console.log('✅ 继续灰度成功');
            this.showSuccess('继续灰度成功');
            
            // 清空表单
            document.getElementById('addNodes').value = '';
            document.getElementById('removeNodes').value = '';

            // 重新加载详情
            setTimeout(() => {
                this.loadDetail();
            }, 1000);

        } catch (error) {
            console.error('❌ 继续灰度失败:', error);
            this.showError('继续灰度失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 完成发布（全量发布或验证完成）
    async doComplete() {
        try {
            this.showLoading();

            const releaseType = this.detailData.releaseType;
            const operationName = releaseType === 'beta' ? '验证完成' : '全量发布';

            console.log(`${operationName}请求:`, this.releaseId);

            const response = await fetch(`/api/v1/release/${this.releaseId}/complete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            console.log(`${operationName}响应状态:`, response.status, response.statusText);

            const responseText = await response.text();
            console.log(`${operationName}响应内容:`, responseText || '(空响应)');

            if (!response.ok) {
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

            // 成功响应：可能返回空响应
            if (responseText && responseText.trim() !== '') {
                try {
                    const data = JSON.parse(responseText);
                    if (data.code && data.code !== 0) {
                        throw new Error(data.desc || `${operationName}失败`);
                    }
                } catch (parseError) {
                    console.warn('解析响应数据失败，但请求已成功:', parseError);
                }
            }

            console.log(`✅ ${operationName}成功`);
            this.showSuccess(`${operationName}成功`);
            
            setTimeout(() => {
                this.loadDetail();
            }, 1000);

        } catch (error) {
            const releaseType = this.detailData.releaseType;
            const operationName = releaseType === 'beta' ? '验证完成' : '全量发布';
            console.error(`❌ ${operationName}失败:`, error);
            this.showError(`${operationName}失败: ${error.message}`);
        } finally {
            this.hideLoading();
        }
    }

    // 回滚（仅用于正式发布）
    async doRollback() {
        try {
            this.showLoading();

            console.log('回滚请求:', this.releaseId);

            const response = await fetch(`/api/v1/release/${this.releaseId}/rollback`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            console.log('回滚响应状态:', response.status, response.statusText);

            const responseText = await response.text();
            console.log('回滚响应内容:', responseText || '(空响应)');

            if (!response.ok) {
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

            // 成功响应：可能返回空响应
            if (responseText && responseText.trim() !== '') {
                try {
                    const data = JSON.parse(responseText);
                    if (data.code && data.code !== 0) {
                        throw new Error(data.desc || '回滚失败');
                    }
                } catch (parseError) {
                    console.warn('解析响应数据失败，但请求已成功:', parseError);
                }
            }

            console.log('✅ 回滚成功');
            this.showSuccess('回滚成功');
            
            setTimeout(() => {
                this.loadDetail();
            }, 1000);

        } catch (error) {
            console.error('❌ 回滚失败:', error);
            this.showError('回滚失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 显示切换模式弹窗
    showSwitchModeModal() {
        // 预填充当前策略
        const policy = this.detailData?.grayPolicy;
        if (policy) {
            if (policy.nodeIds && policy.nodeIds.length > 0) {
                document.querySelector('input[name="switchGrayMode"][value="nodeIds"]').checked = true;
                document.getElementById('switchNodeIds').value = policy.nodeIds.join('\n');
                this.toggleSwitchMode('nodeIds');
            } else if (policy.filter) {
                document.querySelector('input[name="switchGrayMode"][value="filter"]').checked = true;
                
                const filter = policy.filter;
                
                // 设置节点状态
                document.getElementById('switchStatus').value = filter.status || '';
                
                // 设置节点阶段（多选）
                const stagesWrapper = document.getElementById('switchStagesWrapper');
                const stagesCheckboxes = stagesWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]');
                stagesCheckboxes.forEach(checkbox => {
                    checkbox.checked = filter.stages && filter.stages.includes(checkbox.value);
                });
                this.updateMultiSelectText(stagesWrapper);
                this.updateSelectAllState(stagesWrapper);
                
                // 设置业务ID（多选）
                const customerIdsWrapper = document.getElementById('switchCustomerIdsWrapper');
                const customerIdsCheckboxes = customerIdsWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]');
                customerIdsCheckboxes.forEach(checkbox => {
                    checkbox.checked = filter.customerIds && filter.customerIds.includes(parseInt(checkbox.value));
                });
                this.updateMultiSelectText(customerIdsWrapper);
                this.updateSelectAllState(customerIdsWrapper);
                
                this.toggleSwitchMode('filter');
            }
        }

        document.getElementById('switchModeModal').classList.remove('hidden');
    }

    // 隐藏切换模式弹窗
    hideSwitchModeModal() {
        document.getElementById('switchModeModal').classList.add('hidden');
        // 清空表单
        document.getElementById('switchNodeIds').value = '';
        document.getElementById('switchStatus').value = 'online';
        
        // 清空节点阶段多选
        const stagesWrapper = document.getElementById('switchStagesWrapper');
        stagesWrapper.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
        this.updateMultiSelectText(stagesWrapper);
        
        // 清空业务ID多选
        const customerIdsWrapper = document.getElementById('switchCustomerIdsWrapper');
        customerIdsWrapper.querySelectorAll('input[type="checkbox"]').forEach(cb => cb.checked = false);
        this.updateMultiSelectText(customerIdsWrapper);
    }

    // 切换灰度模式
    toggleSwitchMode(mode) {
        const nodeIdsSection = document.getElementById('switchNodeIdsSection');
        const filterSection = document.getElementById('switchFilterSection');

        if (mode === 'nodeIds') {
            nodeIdsSection.style.display = 'block';
            filterSection.style.display = 'none';
        } else {
            nodeIdsSection.style.display = 'none';
            filterSection.style.display = 'block';
        }
    }

    // 处理切换模式
    async handleSwitchMode() {
        const mode = document.querySelector('input[name="switchGrayMode"]:checked').value;
        
        const requestData = {};

        if (mode === 'nodeIds') {
            const nodeIdsText = document.getElementById('switchNodeIds').value.trim();
            if (!nodeIdsText) {
                this.showError('请输入灰度节点');
                return;
            }
            requestData.byNodeIds = true;
        } else {
            // 规则过滤模式
            const filter = {};
            
            // 节点状态
            const status = document.getElementById('switchStatus').value;
            if (status) {
                filter.status = status;
            }
            
            // 节点阶段（多选）
            const stagesWrapper = document.getElementById('switchStagesWrapper');
            const stagesCheckboxes = stagesWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked');
            const stages = Array.from(stagesCheckboxes).map(cb => cb.value);
            if (stages.length > 0) {
                filter.stages = stages;
            }
            
            // 业务ID（多选）
            const customerIdsWrapper = document.getElementById('switchCustomerIdsWrapper');
            const customerIdsCheckboxes = customerIdsWrapper.querySelectorAll('.multiselect-options input[type="checkbox"]:checked');
            const customerIds = Array.from(customerIdsCheckboxes).map(cb => parseInt(cb.value));
            if (customerIds.length > 0) {
                filter.customerIds = customerIds;
            }
            
            if (this.detailData?.deviceType) {
                filter.devType = this.detailData.deviceType;
            }
            
            requestData.byFilter = filter;
        }

        try {
            this.showLoading();

            console.log('切换灰度模式请求:', requestData);

            const response = await fetch(`/api/v1/release/${this.releaseId}/filterswitch`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            console.log('切换灰度模式响应状态:', response.status, response.statusText);

            const responseText = await response.text();
            console.log('切换灰度模式响应内容:', responseText || '(空响应)');

            if (!response.ok) {
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

            // 成功响应：可能返回空响应
            if (responseText && responseText.trim() !== '') {
                try {
                    const data = JSON.parse(responseText);
                    if (data.code && data.code !== 0) {
                        throw new Error(data.desc || '切换灰度模式失败');
                    }
                } catch (parseError) {
                    console.warn('解析响应数据失败，但请求已成功:', parseError);
                }
            }

            console.log('✅ 切换灰度模式成功');
            this.showSuccess('切换灰度模式成功');
            this.hideSwitchModeModal();
            
            setTimeout(() => {
                this.loadDetail();
            }, 1000);

        } catch (error) {
            console.error('❌ 切换灰度模式失败:', error);
            this.showError('切换灰度模式失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 导出节点
    async exportNodes() {
        try {
            this.showLoading();

            const response = await fetch(`/api/v1/release/${this.releaseId}/allownodes/export`, {
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
                throw new Error(data.desc || '导出节点失败');
            }

            const nodes = data.nodes || [];
            
            // 创建下载
            const content = nodes.join('\n');
            const blob = new Blob([content], { type: 'text/plain' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `release_${this.releaseId}_nodes.txt`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);

            this.showSuccess('导出成功');

        } catch (error) {
            console.error('导出节点失败:', error);
            this.showError('导出节点失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 显示操作历史弹窗
    async showHistoryModal() {
        try {
            this.showLoading();

            const response = await fetch(`/api/v1/release/${this.releaseId}/history`, {
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
                throw new Error(data.desc || '获取操作历史失败');
            }

            this.renderHistory(data.items || []);
            document.getElementById('historyModal').classList.remove('hidden');

        } catch (error) {
            console.error('获取操作历史失败:', error);
            this.showError('获取操作历史失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 渲染操作历史
    renderHistory(items) {
        const timeline = document.getElementById('historyTimeline');
        
        if (!items || items.length === 0) {
            timeline.innerHTML = '<div class="empty-state">暂无操作历史</div>';
            return;
        }

        timeline.innerHTML = items.map(item => {
            let detailHtml = '';
            
            // 灰度策略变更
            if (item.grayPolicyInfo) {
                const info = item.grayPolicyInfo;
                detailHtml += '<div class="timeline-detail">';
                
                // 灰度比例变更（重要：放在最前面）
                if (info.afterPercentage !== undefined) {
                    if (info.beforePercentage !== undefined) {
                        detailHtml += `<div><span class="detail-label">灰度比例：</span>${info.beforePercentage}% → ${info.afterPercentage}%</div>`;
                    } else {
                        detailHtml += `<div><span class="detail-label">灰度比例：</span>${info.afterPercentage}%</div>`;
                    }
                }
                
                // 节点变更
                if (info.nodeIdsAdd && info.nodeIdsAdd.length > 0) {
                    const displayNodes = info.nodeIdsAdd.slice(0, 10);
                    const nodeText = displayNodes.join(', ');
                    const moreText = info.nodeIdsAdd.length > 10 ? ` (共${info.nodeIdsAdd.length}个)` : '';
                    detailHtml += `<div><span class="detail-label">新增节点：</span>${nodeText}${moreText}</div>`;
                }
                
                if (info.nodeIdsDel && info.nodeIdsDel.length > 0) {
                    const displayNodes = info.nodeIdsDel.slice(0, 10);
                    const nodeText = displayNodes.join(', ');
                    const moreText = info.nodeIdsDel.length > 10 ? ` (共${info.nodeIdsDel.length}个)` : '';
                    detailHtml += `<div><span class="detail-label">移除节点：</span>${nodeText}${moreText}</div>`;
                }
                
                // 过滤规则变更
                if (info.afterFilter) {
                    detailHtml += `<div><span class="detail-label">灰度模式：</span>规则过滤</div>`;
                    const filter = info.afterFilter;
                    if (filter.status) {
                        detailHtml += `<div><span class="detail-label">节点状态：</span>${filter.status === 'online' ? '在线' : '离线'}</div>`;
                    }
                    if (filter.customerIds && filter.customerIds.length > 0) {
                        detailHtml += `<div><span class="detail-label">客户ID：</span>${filter.customerIds.join(', ')}</div>`;
                    }
                    if (filter.stages && filter.stages.length > 0) {
                        detailHtml += `<div><span class="detail-label">节点阶段：</span>${filter.stages.join(', ')}</div>`;
                    }
                } else if (info.filterChangeMode) {
                    detailHtml += `<div><span class="detail-label">灰度模式：</span>${info.filterChangeMode === 'toNodeIds' ? '切换到指定节点' : '切换到规则过滤'}</div>`;
                }
                
                detailHtml += '</div>';
            }
            
            // 应用配置变更
            if (item.appConfigInfo) {
                const info = item.appConfigInfo;
                detailHtml += '<div class="timeline-detail">';
                
                if (info.beforeMain) {
                    detailHtml += `<div><span class="detail-label">原版本：</span>${this.extractVersionFromUrl(info.beforeMain.url)}</div>`;
                }
                if (info.afterMain) {
                    detailHtml += `<div><span class="detail-label">新版本：</span>${this.extractVersionFromUrl(info.afterMain.url)}</div>`;
                }
                
                detailHtml += '</div>';
            }

            return `
                <div class="timeline-item">
                    <div class="timeline-header">
                        <div class="timeline-operation">${this.getOperationText(item.operation)}</div>
                        <div class="timeline-time">${this.formatDateTime(item.opTime)}</div>
                    </div>
                    <div class="timeline-content">
                        <div class="timeline-meta">
                            操作人：${item.operator || '-'}
                            ${item.beforeState ? ` | 状态变更：${this.getStateText(item.beforeState)} → ${this.getStateText(item.afterState)}` : ''}
                        </div>
                        ${item.remark ? `<div>备注：${item.remark}</div>` : ''}
                        ${detailHtml}
                    </div>
                </div>
            `;
        }).join('');
    }

    // 从URL提取版本信息
    extractVersionFromUrl(url) {
        if (!url) return '-';
        const parts = url.split('/');
        return parts[parts.length - 1] || url;
    }

    // 隐藏操作历史弹窗
    hideHistoryModal() {
        document.getElementById('historyModal').classList.add('hidden');
    }

    // 显示确认弹窗
    showConfirm(title, message, callback) {
        document.getElementById('confirmTitle').textContent = title;
        document.getElementById('confirmMessage').textContent = message;
        this.confirmCallback = callback;
        document.getElementById('confirmModal').classList.remove('hidden');
    }

    // 隐藏确认弹窗
    hideConfirmModal() {
        document.getElementById('confirmModal').classList.add('hidden');
        this.confirmCallback = null;
    }

    // 格式化时间
    formatDateTime(timestamp) {
        if (!timestamp) return '';
        
        const date = new Date(timestamp * 1000);
        
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

    // 获取设备类型文本
    getDeviceTypeText(type) {
        const map = {
            'node': '大节点 x86',
            'smallBox': '小盒子'
        };
        return map[type] || type;
    }

    // 获取操作类型文本
    getOpTypeText(type) {
        const map = {
            'add': '新增',
            'update': '升级',
            'delete': '移除'
        };
        return map[type] || type;
    }

    // 获取发布类型文本
    getReleaseTypeText(type) {
        const map = {
            'formal': '正式发布',
            'beta': '功能验证'
        };
        return map[type] || type;
    }

    // 获取状态文本
    getStateText(state) {
        const map = {
            'created': '已创建',
            'processing': '进行中',
            'complete': '已完成',
            'rollbacked': '已回滚'
        };
        return map[state] || state;
    }

    // 获取包类型文本
    getPackageTypeText(type) {
        const map = {
            'executable': '可执行文件',
            'zipped': '压缩包',
            'tar': 'tar压缩',
            'tar.gz': 'tar.gz压缩'
        };
        return map[type] || type;
    }

    // 获取操作文本
    getOperationText(operation) {
        const map = {
            'create': '创建发布',
            'continue': '继续发布',
            'complete': '完成发布',
            'rollback': '回滚',
            'filterswitch': '切换策略'
        };
        return map[operation] || operation;
    }

    // 显示加载动画
    showLoading() {
        document.getElementById('loading').classList.remove('hidden');
    }

    // 隐藏加载动画
    hideLoading() {
        document.getElementById('loading').classList.add('hidden');
    }

    // 显示错误提示（Toast）
    showError(message) {
        this.showToast(message, 'error');
    }

    // 显示成功提示（Toast）
    showSuccess(message) {
        this.showToast(message, 'success');
    }

    // 显示Toast提示
    showToast(message, type = 'success') {
        // 创建toast元素
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        
        // 图标
        const icon = document.createElement('div');
        icon.className = 'toast-icon';
        icon.textContent = type === 'success' ? '✓' : '✕';
        
        // 消息
        const msg = document.createElement('div');
        msg.className = 'toast-message';
        msg.textContent = message;
        
        toast.appendChild(icon);
        toast.appendChild(msg);
        
        // 添加到页面
        document.body.appendChild(toast);
        
        // 3秒后移除
        setTimeout(() => {
            toast.classList.add('slide-out');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300); // 等待动画完成
        }, 3000);
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    window.releaseDetailManager = new ReleaseDetailManager();
});

