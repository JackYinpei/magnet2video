// API 基础配置
const API_BASE = '/api/v1';
const TORRENT_API = '/api/v1/torrent';
const AUTH_API = '/api/v1/auth';
const USER_API = '/api/v1/user';

// 状态
let currentInfoHash = null;
let parsedTorrent = null;
let progressInterval = null;
let currentUser = null;

// DOM 元素
const elements = {
    // 页面
    pageLogin: document.getElementById('page-login'),
    pageRegister: document.getElementById('page-register'),
    pageLibrary: document.getElementById('page-library'),
    pagePublic: document.getElementById('page-public'),
    pageAdd: document.getElementById('page-add'),
    pageDownloads: document.getElementById('page-downloads'),
    pagePlayer: document.getElementById('page-player'),
    pageProfile: document.getElementById('page-profile'),

    // 导航
    navLinks: document.querySelectorAll('.nav-link'),
    navUser: document.getElementById('nav-user'),

    // 认证表单
    loginForm: document.getElementById('login-form'),
    registerForm: document.getElementById('register-form'),
    loginEmail: document.getElementById('login-email'),
    loginPassword: document.getElementById('login-password'),
    registerEmail: document.getElementById('register-email'),
    registerNickname: document.getElementById('register-nickname'),
    registerPassword: document.getElementById('register-password'),
    registerConfirm: document.getElementById('register-confirm'),
    gotoRegister: document.getElementById('goto-register'),
    gotoLogin: document.getElementById('goto-login'),

    // 添加页面
    magnetInput: document.getElementById('magnet-input'),
    trackerInput: document.getElementById('tracker-input'),
    parseBtn: document.getElementById('parse-btn'),
    fileSelection: document.getElementById('file-selection'),
    torrentName: document.getElementById('torrent-name'),
    torrentSize: document.getElementById('torrent-size'),
    fileList: document.getElementById('file-list'),
    selectAllBtn: document.getElementById('select-all-btn'),
    selectNoneBtn: document.getElementById('select-none-btn'),
    selectVideoBtn: document.getElementById('select-video-btn'),
    downloadBtn: document.getElementById('download-btn'),
    cancelBtn: document.getElementById('cancel-btn'),

    // 媒体库
    libraryGrid: document.getElementById('library-grid'),
    publicGrid: document.getElementById('public-grid'),

    // 下载
    downloadsList: document.getElementById('downloads-list'),

    // 播放器
    backBtn: document.getElementById('back-btn'),
    playerTitle: document.getElementById('player-title'),
    videoPlayer: document.getElementById('video-player'),
    playerFiles: document.getElementById('player-files'),
    playerShare: document.getElementById('player-share'),

    // 个人资料
    profileEmail: document.getElementById('profile-email'),
    profileNickname: document.getElementById('profile-nickname'),
    profileAvatar: document.getElementById('profile-avatar'),
    logoutBtn: document.getElementById('logout-btn'),

    // 通用
    toast: document.getElementById('toast'),
    loading: document.getElementById('loading')
};

// ============ 工具函数 ============

function showLoading() {
    elements.loading.classList.remove('hidden');
}

function hideLoading() {
    elements.loading.classList.add('hidden');
}

function showToast(message, type = 'info') {
    elements.toast.textContent = message;
    elements.toast.className = `toast ${type}`;
    elements.toast.classList.remove('hidden');
    setTimeout(() => {
        elements.toast.classList.add('hidden');
    }, 3000);
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function getStatusText(status) {
    const statusMap = {
        0: '等待中',
        1: '下载中',
        2: '已完成',
        3: '失败',
        4: '已暂停'
    };
    return statusMap[status] || '未知';
}

function isVideoFile(path) {
    const ext = path.toLowerCase().split('.').pop();
    return ['mp4', 'm4v', 'webm', 'mov', 'mkv', 'avi', 'wmv', 'flv'].includes(ext);
}

// ============ Token 管理 ============

function getToken() {
    return localStorage.getItem('auth_token');
}

function setToken(token) {
    localStorage.setItem('auth_token', token);
}

function removeToken() {
    localStorage.removeItem('auth_token');
}

function getAuthHeaders() {
    const token = getToken();
    return token ? { 'Authorization': `Bearer ${token}` } : {};
}

// ============ API 请求 ============

async function apiRequest(url, options = {}) {
    try {
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...getAuthHeaders(),
                ...options.headers
            },
            ...options
        });
        const data = await response.json();

        // 401 未授权，清除token并跳转登录
        if (response.status === 401) {
            removeToken();
            currentUser = null;
            updateNavUser();
            navigateTo('login');
            throw new Error('请先登录');
        }

        if (data.error) {
            throw new Error(data.error.message || '请求失败');
        }
        return data.data;
    } catch (error) {
        throw error;
    }
}

// ============ 认证功能 ============

async function register(email, password, nickname) {
    showLoading();
    try {
        const data = await apiRequest(`${AUTH_API}/register`, {
            method: 'POST',
            body: JSON.stringify({ email, password, nickname })
        });
        setToken(data.token);
        currentUser = data.user;
        updateNavUser();
        showToast('注册成功！', 'success');
        navigateTo('library');
    } catch (error) {
        showToast(error.message || '注册失败', 'error');
    } finally {
        hideLoading();
    }
}

async function login(email, password) {
    showLoading();
    try {
        const data = await apiRequest(`${AUTH_API}/login`, {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        setToken(data.token);
        currentUser = data.user;
        updateNavUser();
        showToast('登录成功！', 'success');
        navigateTo('library');
    } catch (error) {
        showToast(error.message || '登录失败', 'error');
    } finally {
        hideLoading();
    }
}

function logout() {
    removeToken();
    currentUser = null;
    updateNavUser();
    showToast('已退出登录', 'success');
    navigateTo('public');
}

async function loadUserProfile() {
    const token = getToken();
    if (!token) return;

    try {
        const data = await apiRequest(`${USER_API}/profile`);
        currentUser = data.user;
        updateNavUser();
    } catch (error) {
        console.error('加载用户信息失败:', error);
        removeToken();
    }
}

function updateNavUser() {
    if (currentUser) {
        elements.navUser.innerHTML = `
            <div class="user-info" onclick="navigateTo('profile')">
                <div class="user-avatar">${currentUser.nickname?.charAt(0).toUpperCase() || '👤'}</div>
                <span class="user-name">${currentUser.nickname}</span>
            </div>
        `;
    } else {
        elements.navUser.innerHTML = `
            <button class="login-btn" onclick="navigateTo('login')">登录</button>
        `;
    }
}

// ============ 页面导航 ============

function navigateTo(pageName) {
    // 需要登录的页面
    const protectedPages = ['library', 'add', 'downloads', 'profile'];

    if (protectedPages.includes(pageName) && !currentUser) {
        showToast('请先登录', 'error');
        navigateTo('login');
        return;
    }

    // 隐藏所有页面
    document.querySelectorAll('.page').forEach(page => {
        page.classList.remove('active');
    });

    // 更新导航链接
    elements.navLinks.forEach(link => {
        link.classList.remove('active');
        if (link.dataset.page === pageName) {
            link.classList.add('active');
        }
    });

    // 显示目标页面
    const targetPage = document.getElementById(`page-${pageName}`);
    if (targetPage) {
        targetPage.classList.add('active');
    }

    // 页面特定逻辑
    switch (pageName) {
        case 'library':
            loadLibrary();
            break;
        case 'public':
            loadPublicLibrary();
            break;
        case 'downloads':
            loadDownloads();
            startProgressPolling();
            break;
        case 'profile':
            loadProfile();
            break;
        default:
            stopProgressPolling();
    }
}

// ============ 媒体库功能 ============

async function loadLibrary() {
    if (!currentUser) return;

    try {
        const data = await apiRequest(`${TORRENT_API}/list`);
        renderLibrary(data.torrents || [], elements.libraryGrid, true);
    } catch (error) {
        console.error('加载媒体库失败:', error);
    }
}

async function loadPublicLibrary() {
    try {
        const data = await apiRequest(`${TORRENT_API}/public`);
        renderLibrary(data.torrents || [], elements.publicGrid, false);
    } catch (error) {
        console.error('加载公共资源失败:', error);
    }
}

function renderLibrary(torrents, container, isOwner) {
    if (torrents.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>${isOwner ? '暂无媒体，点击"添加"开始下载' : '暂无公开资源'}</p>
            </div>
        `;
        return;
    }

    container.innerHTML = torrents.map(torrent => `
        <div class="poster-card" data-infohash="${torrent.info_hash}">
            <div class="poster-image">
                ${torrent.poster_path
            ? `<img src="${torrent.poster_path}" alt="${torrent.name}">`
            : '🎬'}
            </div>
            <div class="poster-info">
                <div class="poster-title" title="${torrent.name}">${torrent.name}</div>
                <div class="poster-meta">
                    ${formatSize(torrent.total_size)} · ${getStatusText(torrent.status)}
                    ${torrent.is_public ? '<span class="poster-public-badge">公开</span>' : ''}
                </div>
                ${torrent.status !== 2 ? `
                    <div class="poster-progress">
                        <div class="poster-progress-bar" style="width: ${torrent.progress}%"></div>
                    </div>
                ` : ''}
            </div>
        </div>
    `).join('');

    // 绑定点击事件
    container.querySelectorAll('.poster-card').forEach(card => {
        card.addEventListener('click', () => {
            openPlayer(card.dataset.infohash, isOwner);
        });
    });
}

// ============ 下载功能 ============

async function parseMagnet() {
    if (!currentUser) {
        showToast('请先登录', 'error');
        navigateTo('login');
        return;
    }

    const magnetUri = elements.magnetInput.value.trim();
    if (!magnetUri) {
        showToast('请输入磁力链接', 'error');
        return;
    }

    if (!magnetUri.startsWith('magnet:')) {
        showToast('请输入有效的磁力链接', 'error');
        return;
    }

    const trackers = elements.trackerInput.value
        .split('\n')
        .map(t => t.trim())
        .filter(t => t);

    showLoading();

    try {
        const data = await apiRequest(`${TORRENT_API}/parse`, {
            method: 'POST',
            body: JSON.stringify({
                magnet_uri: magnetUri,
                trackers: trackers
            })
        });

        parsedTorrent = data;
        currentInfoHash = data.info_hash;

        // 显示文件选择
        elements.torrentName.textContent = data.name;
        elements.torrentSize.textContent = `总大小: ${formatSize(data.total_size)}`;

        // 渲染文件列表
        renderFileList(data.files);

        elements.fileSelection.classList.remove('hidden');
        showToast('解析成功', 'success');
    } catch (error) {
        showToast(error.message || '解析失败', 'error');
    } finally {
        hideLoading();
    }
}

function renderFileList(files) {
    elements.fileList.innerHTML = files.map((file, index) => `
        <div class="file-item">
            <input type="checkbox" id="file-${index}" data-index="${index}" checked>
            <div class="file-info">
                <div class="file-name">${file.path}</div>
                <div class="file-meta">
                    <span>${file.size_readable || formatSize(file.size)}</span>
                    ${file.is_streamable ? '<span class="streamable">✓ 可播放</span>' : ''}
                </div>
            </div>
        </div>
    `).join('');
}

function getSelectedFiles() {
    const checkboxes = elements.fileList.querySelectorAll('input[type="checkbox"]:checked');
    return Array.from(checkboxes).map(cb => parseInt(cb.dataset.index));
}

function selectAllFiles() {
    elements.fileList.querySelectorAll('input[type="checkbox"]').forEach(cb => {
        cb.checked = true;
    });
}

function selectNoneFiles() {
    elements.fileList.querySelectorAll('input[type="checkbox"]').forEach(cb => {
        cb.checked = false;
    });
}

function selectVideoFiles() {
    const files = parsedTorrent?.files || [];
    elements.fileList.querySelectorAll('input[type="checkbox"]').forEach((cb, index) => {
        cb.checked = isVideoFile(files[index]?.path || '');
    });
}

async function startDownload() {
    if (!currentInfoHash) {
        showToast('请先解析磁力链接', 'error');
        return;
    }

    const selectedFiles = getSelectedFiles();
    if (selectedFiles.length === 0) {
        showToast('请至少选择一个文件', 'error');
        return;
    }

    showLoading();

    try {
        await apiRequest(`${TORRENT_API}/download`, {
            method: 'POST',
            body: JSON.stringify({
                info_hash: currentInfoHash,
                selected_files: selectedFiles,
                trackers: []
            })
        });

        showToast('下载已开始', 'success');
        resetAddForm();
        navigateTo('downloads');
    } catch (error) {
        showToast(error.message || '开始下载失败', 'error');
    } finally {
        hideLoading();
    }
}

function resetAddForm() {
    elements.magnetInput.value = '';
    elements.trackerInput.value = '';
    elements.fileSelection.classList.add('hidden');
    elements.fileList.innerHTML = '';
    parsedTorrent = null;
    currentInfoHash = null;
}

// ============ 下载列表 ============

async function loadDownloads() {
    if (!currentUser) return;

    try {
        const data = await apiRequest(`${TORRENT_API}/list`);
        renderDownloads(data.torrents || []);
    } catch (error) {
        console.error('加载下载列表失败:', error);
    }
}

function renderDownloads(torrents) {
    if (torrents.length === 0) {
        elements.downloadsList.innerHTML = `
            <div class="empty-state">
                <p>暂无下载任务</p>
            </div>
        `;
        return;
    }

    elements.downloadsList.innerHTML = torrents.map(torrent => `
        <div class="download-item" data-infohash="${torrent.info_hash}">
            <div class="download-header">
                <div class="download-name">${torrent.name}</div>
                <div class="download-actions">
                    ${torrent.status === 1 ? `
                        <button class="btn btn-sm pause-btn" data-infohash="${torrent.info_hash}">暂停</button>
                    ` : torrent.status === 4 ? `
                        <button class="btn btn-sm resume-btn" data-infohash="${torrent.info_hash}">继续</button>
                    ` : ''}
                    <button class="btn btn-sm ${torrent.is_public ? 'btn-success' : ''}" 
                            onclick="togglePublic('${torrent.info_hash}', ${!torrent.is_public})">
                        ${torrent.is_public ? '✓ 公开' : '设为公开'}
                    </button>
                    <button class="btn btn-sm remove-btn" data-infohash="${torrent.info_hash}">删除</button>
                </div>
            </div>
            <div class="download-progress">
                <div class="download-progress-bar ${torrent.status === 2 ? 'completed' : ''}" 
                     style="width: ${torrent.progress}%"></div>
            </div>
            <div class="download-stats">
                <div class="download-stat">进度: <span>${torrent.progress.toFixed(1)}%</span></div>
                <div class="download-stat">大小: <span>${formatSize(torrent.total_size)}</span></div>
                <div class="download-stat">速度: <span>${torrent.download_speed_readable || '0 B/s'}</span></div>
                <div class="download-stat">状态: <span>${getStatusText(torrent.status)}</span></div>
            </div>
        </div>
    `).join('');

    // 绑定按钮事件
    elements.downloadsList.querySelectorAll('.pause-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            pauseDownload(btn.dataset.infohash);
        });
    });

    elements.downloadsList.querySelectorAll('.resume-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            resumeDownload(btn.dataset.infohash);
        });
    });

    elements.downloadsList.querySelectorAll('.remove-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            removeTorrent(btn.dataset.infohash);
        });
    });
}

async function pauseDownload(infoHash) {
    try {
        await apiRequest(`${TORRENT_API}/pause`, {
            method: 'POST',
            body: JSON.stringify({ info_hash: infoHash })
        });
        showToast('已暂停', 'success');
        loadDownloads();
    } catch (error) {
        showToast(error.message || '暂停失败', 'error');
    }
}

async function resumeDownload(infoHash) {
    try {
        await apiRequest(`${TORRENT_API}/resume`, {
            method: 'POST',
            body: JSON.stringify({
                info_hash: infoHash,
                selected_files: []
            })
        });
        showToast('已继续', 'success');
        loadDownloads();
    } catch (error) {
        showToast(error.message || '继续失败', 'error');
    }
}

async function removeTorrent(infoHash) {
    if (!confirm('确定要删除这个任务吗？')) {
        return;
    }

    try {
        await apiRequest(`${TORRENT_API}/remove`, {
            method: 'POST',
            body: JSON.stringify({
                info_hash: infoHash,
                delete_files: true
            })
        });
        showToast('已删除', 'success');
        loadDownloads();
        loadLibrary();
    } catch (error) {
        showToast(error.message || '删除失败', 'error');
    }
}

async function togglePublic(infoHash, isPublic) {
    try {
        await apiRequest(`${USER_API}/torrent/public`, {
            method: 'POST',
            body: JSON.stringify({
                info_hash: infoHash,
                is_public: isPublic
            })
        });
        showToast(isPublic ? '已设为公开' : '已设为私有', 'success');
        loadDownloads();
    } catch (error) {
        showToast(error.message || '设置失败', 'error');
    }
}

function startProgressPolling() {
    stopProgressPolling();
    progressInterval = setInterval(() => {
        loadDownloads();
    }, 3000);
}

function stopProgressPolling() {
    if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
    }
}

// ============ 播放器 ============

let currentTorrentIsOwner = false;

async function openPlayer(infoHash, isOwner = false) {
    showLoading();
    currentTorrentIsOwner = isOwner;

    try {
        const data = await apiRequest(`${TORRENT_API}/detail/${infoHash}`);

        currentInfoHash = infoHash;
        elements.playerTitle.textContent = data.name;

        // 渲染分享按钮（仅对自己的资源显示）
        if (isOwner) {
            elements.playerShare.innerHTML = `
                <div class="share-toggle ${data.is_public ? 'public' : 'private'}" 
                     onclick="togglePublicFromPlayer('${infoHash}', ${!data.is_public})">
                    ${data.is_public ? '✓ 已公开分享' : '🔒 设为公开'}
                </div>
            `;
        } else {
            elements.playerShare.innerHTML = '';
        }

        // 渲染文件列表
        const videoFiles = data.files.filter(f => isVideoFile(f.path));

        if (videoFiles.length === 0) {
            elements.playerFiles.innerHTML = '<p>暂无可播放的视频文件</p>';
        } else {
            elements.playerFiles.innerHTML = `
                <h4>文件列表</h4>
                ${data.files.map((file, index) => {
                const isVideo = isVideoFile(file.path);
                return `
                        <div class="player-file-item ${!isVideo || !file.is_streamable ? 'disabled' : ''}" 
                             data-index="${index}" 
                             data-path="${file.path}"
                             data-streamable="${file.is_streamable}">
                            <span>${file.path}</span>
                            <span>${file.size_readable || formatSize(file.size)}</span>
                        </div>
                    `;
            }).join('')}
            `;

            // 绑定文件点击事件
            elements.playerFiles.querySelectorAll('.player-file-item:not(.disabled)').forEach(item => {
                item.addEventListener('click', () => {
                    playFile(item.dataset.path);
                    // 更新选中状态
                    elements.playerFiles.querySelectorAll('.player-file-item').forEach(i => {
                        i.classList.remove('active');
                    });
                    item.classList.add('active');
                });
            });

            // 自动播放第一个可播放的文件
            const firstPlayable = data.files.find(f => isVideoFile(f.path) && f.is_streamable);
            if (firstPlayable) {
                playFile(firstPlayable.path);
            }
        }

        // 隐藏导航栏的 active 状态
        elements.navLinks.forEach(link => link.classList.remove('active'));

        // 显示播放器页面
        document.querySelectorAll('.page').forEach(page => page.classList.remove('active'));
        elements.pagePlayer.classList.add('active');

    } catch (error) {
        showToast(error.message || '加载失败', 'error');
    } finally {
        hideLoading();
    }
}

async function togglePublicFromPlayer(infoHash, isPublic) {
    await togglePublic(infoHash, isPublic);
    // 刷新分享按钮状态
    elements.playerShare.innerHTML = `
        <div class="share-toggle ${isPublic ? 'public' : 'private'}" 
             onclick="togglePublicFromPlayer('${infoHash}', ${!isPublic})">
            ${isPublic ? '✓ 已公开分享' : '🔒 设为公开'}
        </div>
    `;
}

function playFile(filePath) {
    const videoUrl = `${TORRENT_API}/file/${currentInfoHash}/${encodeURIComponent(filePath)}`;
    elements.videoPlayer.src = videoUrl;
    elements.videoPlayer.play().catch(e => {
        console.log('Auto-play prevented:', e);
    });
}

// ============ 个人资料 ============

function loadProfile() {
    if (!currentUser) return;

    elements.profileEmail.textContent = currentUser.email;
    elements.profileNickname.textContent = currentUser.nickname;
    elements.profileAvatar.textContent = currentUser.nickname?.charAt(0).toUpperCase() || '👤';
}

// ============ 事件监听初始化 ============

function initEventListeners() {
    // 导航
    elements.navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            navigateTo(link.dataset.page);
        });
    });

    // 登录表单
    elements.loginForm.addEventListener('submit', (e) => {
        e.preventDefault();
        login(elements.loginEmail.value, elements.loginPassword.value);
    });

    // 注册表单
    elements.registerForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const password = elements.registerPassword.value;
        const confirm = elements.registerConfirm.value;

        if (password !== confirm) {
            showToast('两次输入的密码不一致', 'error');
            return;
        }

        register(
            elements.registerEmail.value,
            password,
            elements.registerNickname.value
        );
    });

    // 切换登录/注册
    elements.gotoRegister.addEventListener('click', (e) => {
        e.preventDefault();
        document.querySelectorAll('.page').forEach(page => page.classList.remove('active'));
        elements.pageRegister.classList.add('active');
    });

    elements.gotoLogin.addEventListener('click', (e) => {
        e.preventDefault();
        document.querySelectorAll('.page').forEach(page => page.classList.remove('active'));
        elements.pageLogin.classList.add('active');
    });

    // 解析按钮
    elements.parseBtn.addEventListener('click', parseMagnet);

    // 文件选择按钮
    elements.selectAllBtn.addEventListener('click', selectAllFiles);
    elements.selectNoneBtn.addEventListener('click', selectNoneFiles);
    elements.selectVideoBtn.addEventListener('click', selectVideoFiles);

    // 下载和取消按钮
    elements.downloadBtn.addEventListener('click', startDownload);
    elements.cancelBtn.addEventListener('click', resetAddForm);

    // 返回按钮
    elements.backBtn.addEventListener('click', () => {
        elements.videoPlayer.pause();
        elements.videoPlayer.src = '';
        if (currentTorrentIsOwner) {
            navigateTo('library');
        } else {
            navigateTo('public');
        }
    });

    // 退出登录
    elements.logoutBtn.addEventListener('click', logout);

    // 键盘快捷键
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && elements.pagePlayer.classList.contains('active')) {
            elements.videoPlayer.pause();
            elements.videoPlayer.src = '';
            navigateTo('library');
        }
    });
}

// ============ 初始化 ============

async function init() {
    initEventListeners();

    // 尝试加载用户信息
    await loadUserProfile();

    // 根据登录状态显示不同页面
    if (currentUser) {
        navigateTo('library');
    } else {
        navigateTo('public');
    }
}

// 启动应用
document.addEventListener('DOMContentLoaded', init);

// 暴露全局函数供onclick使用
window.navigateTo = navigateTo;
window.togglePublic = togglePublic;
window.togglePublicFromPlayer = togglePublicFromPlayer;
