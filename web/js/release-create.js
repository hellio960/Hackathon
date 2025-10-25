class ReleaseCreateManager {
    constructor() {
        this.currentStep = 1;
        this.formData = {
            step1: {},
            step2: {}
        };
        this.currentConfig = null; // 当前版本配置
        this.packagesList = []; // 可用包列表
        this.customPath = null; // 自定义路径
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadComponents();
    }

    bindEvents() {
        // 步骤1 - 下一步按钮
        document.getElementById('nextStepBtn').addEventListener('click', () => {
            this.handleNextStep();
        });

        // 步骤2 - 返回按钮
        document.getElementById('prevStepBtn').addEventListener('click', () => {
            this.handlePrevStep();
        });

        // 步骤2 - 提交按钮
        document.getElementById('submitBtn').addEventListener('click', () => {
            this.handleSubmit();
        });

        // 组件和设备类型变化时，加载当前配置和包列表
        document.getElementById('appName').addEventListener('change', () => {
            this.handleAppOrDevTypeChange();
        });

        document.getElementById('devType').addEventListener('change', () => {
            this.handleAppOrDevTypeChange();
        });

        // 操作类型变化
        document.getElementById('opType').addEventListener('change', (e) => {
            this.handleOpTypeChange(e.target.value);
        });

        // 灰度模式切换
        document.querySelectorAll('input[name="grayMode"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.toggleGrayMode(e.target.value);
            });
        });

        // 指定路径筛选
        document.getElementById('specifyPathBtn').addEventListener('click', () => {
            this.showPathFilterModal();
        });

        document.getElementById('closePathFilterModal').addEventListener('click', () => {
            this.hidePathFilterModal();
        });

        document.getElementById('cancelPathFilterBtn').addEventListener('click', () => {
            this.hidePathFilterModal();
        });

        document.getElementById('confirmPathFilterBtn').addEventListener('click', () => {
            this.handlePathFilter();
        });

        // 包选择变化
        document.getElementById('packageSelect').addEventListener('change', (e) => {
            this.handlePackageChange(e.target.value);
        });

        // 点击弹窗外部关闭
        document.getElementById('pathFilterModal').addEventListener('click', (e) => {
            if (e.target.id === 'pathFilterModal') {
                this.hidePathFilterModal();
            }
        });

        // 初始化节点阶段多选下拉框（点击切换选中状态）
        this.initMultiSelect('stages');
    }

    // 初始化多选下拉框，支持点击切换选中状态
    initMultiSelect(selectId) {
        const select = document.getElementById(selectId);
        if (!select) return;

        select.addEventListener('mousedown', function(e) {
            e.preventDefault();
            
            const option = e.target;
            if (option.tagName === 'OPTION') {
                // 切换选中状态
                option.selected = !option.selected;
                
                // 触发change事件（如果需要）
                select.dispatchEvent(new Event('change'));
            }
            
            // 保持焦点在select上
            select.focus();
        });

        // 阻止鼠标抬起时的默认行为
        select.addEventListener('mouseup', function(e) {
            e.preventDefault();
        });

        // 阻止点击时的默认行为
        select.addEventListener('click', function(e) {
            e.preventDefault();
        });
    }

    // 加载组件列表
    async loadComponents() {
        try {
            this.showLoading();
            
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
        } finally {
            this.hideLoading();
        }
    }

    // 渲染组件选项
    renderComponentOptions(apps) {
        const select = document.getElementById('appName');
        const currentValue = select.value;
        
        // 清空现有选项（保留第一个placeholder）
        select.innerHTML = '<option value="">请选择组件</option>';
        
        // 去重
        const uniqueApps = [...new Set(apps.map(app => app.name))];
        
        uniqueApps.forEach(appName => {
            const option = document.createElement('option');
            option.value = appName;
            option.textContent = appName;
            select.appendChild(option);
        });

        // 恢复之前的选择
        if (currentValue) {
            select.value = currentValue;
        }
    }

    // 处理组件或设备类型变化
    async handleAppOrDevTypeChange() {
        const appName = document.getElementById('appName').value;
        const devType = document.getElementById('devType').value;

        if (!appName || !devType) {
            return;
        }

        // 加载当前配置和包列表
        await Promise.all([
            this.loadCurrentConfig(appName, devType),
            this.loadPackages(appName, devType)
        ]);
    }

    // 加载当前版本配置
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
            // 从返回的配置中提取指定应用的配置
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

    // 加载可用包列表
    async loadPackages(appName, devType, path = null) {
        try {
            this.showLoading();

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
            this.hideLoading();
        }
    }

    // 渲染包选项
    renderPackageOptions() {
        const select = document.getElementById('packageSelect');
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

        // 自动填充MD5和URL
        document.getElementById('cmd').value = this.extractProgramName(pkg.file);
        
        // 根据文件扩展名推测包类型
        const packageType = this.guessPackageType(pkg.file);
        document.getElementById('packageType').value = packageType;
    }

    // 从文件名提取程序名
    extractProgramName(filename) {
        // 移除路径和扩展名
        return filename.replace(/.*\//, '').replace(/\.(tar\.gz|tar|zip|tgz)$/, '');
    }

    // 推测包类型
    guessPackageType(filename) {
        if (filename.match(/\.(tar\.gz|tar|zip|tgz)$/)) {
            return 'zipped';
        }
        return 'executable';
    }

    // 操作类型变化处理
    handleOpTypeChange(opType) {
        const currentConfigSection = document.getElementById('currentConfigSection');
        
        if (opType === 'add') {
            // 新增组件：不显示当前版本
            currentConfigSection.style.display = 'none';
        } else {
            // 升级或移除：显示当前版本
            currentConfigSection.style.display = 'block';
        }
    }

    // 显示第一步
    showStep1() {
        document.getElementById('step1').classList.add('active');
        document.getElementById('step2').classList.remove('active');
        this.currentStep = 1;
    }

    // 显示第二步
    showStep2() {
        document.getElementById('step1').classList.remove('active');
        document.getElementById('step2').classList.add('active');
        this.currentStep = 2;
    }

    // 下一步
    async handleNextStep() {
        // 验证步骤1
        if (!this.validateStep1()) {
            return;
        }

        // 保存步骤1数据
        this.formData.step1 = {
            appName: document.getElementById('appName').value,
            devType: document.getElementById('devType').value,
            releaseType: document.querySelector('input[name="releaseType"]:checked').value,
            opType: document.getElementById('opType').value
        };

        // 显示步骤2
        this.showStep2();

        // 填充基本信息回显
        this.displayStep1Info();

        // 填充当前版本配置
        this.displayCurrentConfig();

        // 如果存在当前配置，预填充发布版本字段
        this.prefillReleaseFields();
    }

    // 验证步骤1
    validateStep1() {
        const appName = document.getElementById('appName').value;
        const devType = document.getElementById('devType').value;
        const opType = document.getElementById('opType').value;

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

    // 显示步骤1信息
    displayStep1Info() {
        const data = this.formData.step1;
        
        document.getElementById('displayAppName').textContent = data.appName;
        document.getElementById('displayDevType').textContent = this.getDevTypeText(data.devType);
        document.getElementById('displayReleaseType').textContent = this.getReleaseTypeText(data.releaseType);
    }

    // 显示当前版本配置
    displayCurrentConfig() {
        const opType = this.formData.step1.opType;
        
        if (opType === 'add' || !this.currentConfig) {
            // 新增组件或无当前配置
            document.getElementById('currentUrl').value = '';
            document.getElementById('currentMd5').value = '';
            document.getElementById('currentPackageType').value = '';
            document.getElementById('currentCmd').value = '';
            document.getElementById('currentArgs').value = '';
            document.getElementById('currentWorkDir').value = '';
            
            if (opType === 'add') {
                document.getElementById('currentConfigSection').style.display = 'none';
            }
        } else {
            // 升级或移除组件，显示当前配置
            document.getElementById('currentConfigSection').style.display = 'block';
            document.getElementById('currentUrl').value = this.currentConfig.url || '';
            document.getElementById('currentMd5').value = this.currentConfig.md5 || '';
            document.getElementById('currentPackageType').value = this.currentConfig.type || '';
            document.getElementById('currentCmd').value = this.currentConfig.cmd || '';
            document.getElementById('currentArgs').value = (this.currentConfig.args || []).join(' ');
            document.getElementById('currentWorkDir').value = this.currentConfig.dir || '';
        }
    }

    // 预填充发布版本字段（基于当前配置）
    prefillReleaseFields() {
        const opType = this.formData.step1.opType;

        // 只在升级或删除操作时，且存在当前配置时预填充
        if ((opType === 'update' || opType === 'delete') && this.currentConfig) {
            console.log('预填充发布版本字段:', this.currentConfig);

            // 包类型
            if (this.currentConfig.type) {
                document.getElementById('packageType').value = this.currentConfig.type;
            }

            // 程序名
            if (this.currentConfig.cmd) {
                document.getElementById('cmd').value = this.currentConfig.cmd;
            }

            // 工作目录
            if (this.currentConfig.dir) {
                document.getElementById('workDir').value = this.currentConfig.dir;
            }

            // 启动参数（数组转为空格分隔的字符串）
            if (this.currentConfig.args && Array.isArray(this.currentConfig.args)) {
                document.getElementById('args').value = this.currentConfig.args.join(' ');
            } else if (this.currentConfig.args) {
                // 如果不是数组，直接设置
                document.getElementById('args').value = this.currentConfig.args;
            }

            console.log('✅ 发布版本字段预填充完成');
        } else if (opType === 'add') {
            // 新增操作，清空这些字段
            document.getElementById('packageType').value = '';
            document.getElementById('cmd').value = '';
            document.getElementById('workDir').value = '';
            document.getElementById('args').value = '';
        }
    }

    // 返回上一步
    handlePrevStep() {
        this.showStep1();
    }

    // 提交表单
    async handleSubmit() {
        // 验证步骤2
        if (!this.validateStep2()) {
            return;
        }

        // 构造请求数据
        const requestData = this.buildRequestData();

        try {
            this.showLoading();

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
            
            // 延迟跳转到发布列表页
            setTimeout(() => {
                window.location.href = 'release-list.html';
            }, 1500);

        } catch (error) {
            console.error('❌ 创建发布任务失败:', error);
            this.showError('创建发布任务失败: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    // 验证步骤2
    validateStep2() {
        const opType = this.formData.step1.opType;
        
        // 如果不是移除操作，需要验证发布版本配置
        if (opType !== 'delete') {
            const packageSelect = document.getElementById('packageSelect').value;
            const packageType = document.getElementById('packageType').value;
            const cmd = document.getElementById('cmd').value.trim();
            const workDir = document.getElementById('workDir').value.trim();

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

        // 验证灰度策略
        const grayMode = document.querySelector('input[name="grayMode"]:checked').value;
        
        if (grayMode === 'nodeIds') {
            const nodeIds = document.getElementById('nodeIds').value.trim();
            if (!nodeIds) {
                this.showError('请输入灰度节点');
                return false;
            }
        } else {
            // 规则过滤模式暂时不强制校验
        }

        const percentage = document.getElementById('percentage').value;
        if (!percentage || percentage < 0 || percentage > 100) {
            this.showError('请输入有效的灰度比例（0-100）');
            return false;
        }

        const describe = document.getElementById('describe').value.trim();
        if (!describe) {
            this.showError('请输入发布描述');
            return false;
        }

        return true;
    }

    // 构造请求数据
    buildRequestData() {
        const step1 = this.formData.step1;
        const opType = step1.opType;
        
        const requestData = {
            devType: step1.devType,
            appName: step1.appName,
            describe: document.getElementById('describe').value.trim(),
            releaseType: step1.releaseType,
            opType: opType,
            grayPolicy: this.buildGrayPolicy()
        };

        // 如果不是移除操作，添加应用配置
        if (opType !== 'delete') {
            requestData.appConfig = this.buildAppConfig();
        }

        return requestData;
    }

    // 构造应用配置
    buildAppConfig() {
        const packageIndex = parseInt(document.getElementById('packageSelect').value);
        const pkg = this.packagesList[packageIndex];
        
        const args = document.getElementById('args').value.trim();
        const argsArray = args ? args.split(/\s+/) : [];

        return {
            url: pkg.url,
            type: document.getElementById('packageType').value,
            cmd: document.getElementById('cmd').value.trim(),
            args: argsArray,
            dir: document.getElementById('workDir').value.trim(),
            healthUrl: '',
            md5: pkg.md5
        };
    }

    // 构造灰度策略
    buildGrayPolicy() {
        const grayMode = document.querySelector('input[name="grayMode"]:checked').value;
        const percentage = parseInt(document.getElementById('percentage').value);
        
        const policy = {
            percentage: percentage,
            statInfo: {
                totalCount: 0
            }
        };

        if (grayMode === 'nodeIds') {
            // 灰度节点模式
            const nodeIdsText = document.getElementById('nodeIds').value.trim();
            policy.nodeIds = nodeIdsText.split('\n').map(id => id.trim()).filter(id => id);
        } else {
            // 规则过滤模式
            const filter = {};
            
            // 节点ID（换行分隔）
            const filterNodeIds = document.getElementById('filterNodeIds').value.trim();
            if (filterNodeIds) {
                filter.nodeIds = filterNodeIds.split('\n').map(id => id.trim()).filter(id => id);
            }
            
            // 节点状态（必填字段，即使为空也要设置）
            const status = document.getElementById('nodeStatus').value;
            filter.status = status || '';
            
            // 节点阶段（必填字段，即使为空数组也要设置）
            const stageSelect = document.getElementById('stages');
            const stages = Array.from(stageSelect.selectedOptions).map(option => option.value);
            filter.stages = stages;
            
            filter.devType = this.formData.step1.devType;
            
            policy.filter = filter;
        }

        return policy;
    }

    // 切换灰度模式
    toggleGrayMode(mode) {
        const nodeIdsSection = document.getElementById('nodeIdsSection');
        const filterSection = document.getElementById('filterSection');

        if (mode === 'nodeIds') {
            nodeIdsSection.style.display = 'block';
            filterSection.style.display = 'none';
        } else {
            nodeIdsSection.style.display = 'none';
            filterSection.style.display = 'block';
        }
    }

    // 显示路径筛选弹窗
    showPathFilterModal() {
        document.getElementById('pathFilterModal').classList.remove('hidden');
    }

    // 隐藏路径筛选弹窗
    hidePathFilterModal() {
        document.getElementById('pathFilterModal').classList.add('hidden');
        document.getElementById('customPath').value = '';
    }

    // 处理路径筛选
    async handlePathFilter() {
        const path = document.getElementById('customPath').value.trim();
        
        if (!path) {
            this.showError('请输入包路径');
            return;
        }

        const appName = this.formData.step1.appName;
        const devType = this.formData.step1.devType;

        this.customPath = path;
        this.hidePathFilterModal();

        // 重新加载包列表
        await this.loadPackages(appName, devType, path);
    }

    // 获取设备类型文本
    getDevTypeText(devType) {
        const map = {
            'node': '大节点 x86',
            'smallBox': '小盒子'
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

    // 显示加载动画
    showLoading() {
        document.getElementById('loading').classList.remove('hidden');
    }

    // 隐藏加载动画
    hideLoading() {
        document.getElementById('loading').classList.add('hidden');
    }

    // 显示错误提示
    showError(message) {
        alert('错误: ' + message);
    }

    // 显示成功提示
    showSuccess(message) {
        alert('成功: ' + message);
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
    window.releaseCreateManager = new ReleaseCreateManager();
});

