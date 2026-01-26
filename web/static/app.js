// API 基础配置
const API_BASE = '/api/v1';
const TORRENT_API = '/api/v1/torrent';
const AUTH_API = '/api/v1/auth';
const USER_API = '/api/v1/user';
const ADMIN_API = '/api/v1/admin';

// 状态
let currentInfoHash = null;
let parsedTorrent = null;
let progressInterval = null;
let currentUser = null;

// Admin pagination state
let adminUsersPage = 1;
let adminResourcesPage = 1;
const adminPageSize = 10;

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
    pageAdmin: document.getElementById('page-admin'),

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

function isSubtitleFile(path) {
    const ext = path.toLowerCase().split('.').pop();
    return ['srt', 'vtt', 'ass', 'ssa'].includes(ext);
}

// Convert SRT to WebVTT format
function srtToVtt(srtContent) {
    // Add WebVTT header
    let vtt = 'WEBVTT\n\n';

    // Normalize line endings
    let content = srtContent.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

    // Split into blocks
    const blocks = content.trim().split(/\n\n+/);

    for (const block of blocks) {
        const lines = block.split('\n');
        if (lines.length < 2) continue;

        // Find the timestamp line (contains -->)
        let timestampLineIndex = -1;
        for (let i = 0; i < lines.length; i++) {
            if (lines[i].includes('-->')) {
                timestampLineIndex = i;
                break;
            }
        }

        if (timestampLineIndex === -1) continue;

        // Convert timestamps from SRT format (00:00:00,000) to VTT format (00:00:00.000)
        const timestampLine = lines[timestampLineIndex].replace(/,/g, '.');

        // Get subtitle text (everything after timestamp line)
        const subtitleText = lines.slice(timestampLineIndex + 1).join('\n');

        if (subtitleText.trim()) {
            vtt += timestampLine + '\n';
            vtt += subtitleText + '\n\n';
        }
    }

    return vtt;
}

// Fetch and convert subtitle file to VTT blob URL
async function loadSubtitle(subtitleUrl, subtitlePath) {
    try {
        const response = await fetch(subtitleUrl, { headers: getAuthHeaders() });
        if (!response.ok) {
            throw new Error('Failed to fetch subtitle');
        }

        let content = await response.text();
        const ext = subtitlePath.toLowerCase().split('.').pop();

        // Convert to VTT if needed
        if (ext === 'srt') {
            content = srtToVtt(content);
        } else if (ext === 'ass' || ext === 'ssa') {
            // Basic ASS/SSA to VTT conversion (simplified)
            content = assToVtt(content);
        }
        // VTT files don't need conversion

        // Create blob URL for the VTT content
        const blob = new Blob([content], { type: 'text/vtt' });
        return URL.createObjectURL(blob);
    } catch (error) {
        console.error('Error loading subtitle:', error);
        return null;
    }
}

// Basic ASS/SSA to VTT conversion
function assToVtt(assContent) {
    let vtt = 'WEBVTT\n\n';

    const lines = assContent.split(/\r?\n/);
    const dialogueRegex = /^Dialogue:\s*\d+,([^,]+),([^,]+),[^,]*,[^,]*,[^,]*,[^,]*,[^,]*,[^,]*,(.*)$/;

    for (const line of lines) {
        const match = line.match(dialogueRegex);
        if (match) {
            // Convert ASS timestamp (0:00:00.00) to VTT format (00:00:00.000)
            const startTime = convertAssTime(match[1]);
            const endTime = convertAssTime(match[2]);
            let text = match[3];

            // Remove ASS styling tags
            text = text.replace(/\{[^}]*\}/g, '');
            // Convert \N to newline
            text = text.replace(/\\N/gi, '\n');

            if (text.trim()) {
                vtt += `${startTime} --> ${endTime}\n`;
                vtt += `${text}\n\n`;
            }
        }
    }

    return vtt;
}

// Convert ASS timestamp to VTT format
function convertAssTime(assTime) {
    // ASS format: H:MM:SS.cc (centiseconds)
    // VTT format: HH:MM:SS.mmm (milliseconds)
    const parts = assTime.trim().split(':');
    if (parts.length !== 3) return '00:00:00.000';

    const hours = parts[0].padStart(2, '0');
    const minutes = parts[1].padStart(2, '0');
    const secParts = parts[2].split('.');
    const seconds = secParts[0].padStart(2, '0');
    const centiseconds = secParts[1] || '00';
    const milliseconds = (parseInt(centiseconds) * 10).toString().padStart(3, '0');

    return `${hours}:${minutes}:${seconds}.${milliseconds}`;
}

// Find matching subtitle for a video file
function findMatchingSubtitle(videoPath, subtitleFiles) {
    const videoBaseName = videoPath.replace(/\.[^/.]+$/, '').toLowerCase();

    // First, try to find exact match (same base name)
    for (const sub of subtitleFiles) {
        const subBaseName = sub.path.replace(/\.[^/.]+$/, '').toLowerCase();
        if (subBaseName === videoBaseName) {
            return sub;
        }
    }

    // Try to find partial match
    for (const sub of subtitleFiles) {
        const subBaseName = sub.path.replace(/\.[^/.]+$/, '').toLowerCase();
        if (videoBaseName.includes(subBaseName) || subBaseName.includes(videoBaseName)) {
            return sub;
        }
    }

    return null;
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
    // Show/hide admin link based on role
    const adminLink = document.querySelector('.nav-link.admin-only');
    if (adminLink) {
        if (currentUser && currentUser.role === 'admin') {
            adminLink.classList.remove('hidden');
        } else {
            adminLink.classList.add('hidden');
        }
    }

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
    const protectedPages = ['library', 'add', 'downloads', 'profile', 'admin'];

    if (protectedPages.includes(pageName) && !currentUser) {
        showToast('请先登录', 'error');
        navigateTo('login');
        return;
    }

    // Admin page requires admin role
    if (pageName === 'admin' && currentUser?.role !== 'admin') {
        showToast('需要管理员权限', 'error');
        navigateTo('library');
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
        case 'admin':
            loadAdminPage();
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
            ${renderTranscodeStatus(torrent)}
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

// Render transcode status for a torrent
function renderTranscodeStatus(torrent) {
    // Show "待检测" for downloading torrents with transcode_status = 0
    if (torrent.transcode_status === 0 && torrent.status !== 2) {
        return `
            <div class="download-transcode">
                <div class="download-transcode-label">
                    转码状态: <span class="transcode-badge pending">待检测</span>
                </div>
            </div>
        `;
    }

    // Only show if transcode_status exists and is not 0 (no transcode needed)
    if (!torrent.transcode_status || torrent.transcode_status === 0) {
        return '';
    }

    const statusText = getTranscodeText(torrent.transcode_status, torrent.status);
    const statusClass = getTranscodeClass(torrent.transcode_status);
    const progress = torrent.transcode_progress || 0;
    const transcoded = torrent.transcoded_count || 0;
    const total = torrent.total_transcode || 0;

    let progressHtml = '';
    if (torrent.transcode_status === 2) { // Processing
        progressHtml = `
            <div class="transcode-progress">
                <div class="transcode-progress-bar">
                    <div class="transcode-progress-fill" style="width: ${progress}%"></div>
                </div>
                <span class="transcode-progress-text">${progress}%</span>
            </div>
        `;
    }

    return `
        <div class="download-transcode">
            <div class="download-transcode-label">
                转码状态: <span class="transcode-badge ${statusClass}">${statusText}</span>
                ${total > 0 ? `(${transcoded}/${total} 文件)` : ''}
            </div>
            ${progressHtml}
        </div>
    `;
}

// ============ 播放器 ============

let currentTorrentIsOwner = false;
let currentTorrentFiles = [];
let currentSubtitleFiles = [];
let currentSubtitleBlobUrl = null;

async function openPlayer(infoHash, isOwner = false) {
    showLoading();
    currentTorrentIsOwner = isOwner;

    // Clean up previous subtitle blob URL
    if (currentSubtitleBlobUrl) {
        URL.revokeObjectURL(currentSubtitleBlobUrl);
        currentSubtitleBlobUrl = null;
    }

    try {
        const data = await apiRequest(`${TORRENT_API}/detail/${infoHash}`);

        currentInfoHash = infoHash;
        currentTorrentFiles = data.files;
        currentSubtitleFiles = data.files.filter(f => isSubtitleFile(f.path));
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
            // Build subtitle selector HTML
            let subtitleSelectorHtml = '';
            if (currentSubtitleFiles.length > 0) {
                subtitleSelectorHtml = `
                    <div class="subtitle-selector">
                        <h4>📝 字幕</h4>
                        <div class="subtitle-options">
                            <div class="subtitle-option active" data-path="" onclick="selectSubtitle('')">
                                <span>关闭字幕</span>
                            </div>
                            ${currentSubtitleFiles.map(sub => `
                                <div class="subtitle-option" data-path="${sub.path}" onclick="selectSubtitle('${sub.path}')">
                                    <span>${sub.path}</span>
                                    <span class="subtitle-size">${sub.size_readable || formatSize(sub.size)}</span>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }

            elements.playerFiles.innerHTML = `
                ${subtitleSelectorHtml}
                <h4>文件列表</h4>
                ${data.files.map((file, index) => {
                const isVideo = isVideoFile(file.path);
                const isSubtitle = isSubtitleFile(file.path);
                const icon = isVideo ? '🎬' : (isSubtitle ? '📝' : '📄');
                const hasTranscoded = file.transcode_status === 3 && file.transcoded_path;
                const transcodeStatusText = getFileTranscodeStatus(file);
                // File is playable if: (video AND streamable) OR has transcoded version
                const isPlayable = isVideo && (file.is_streamable || hasTranscoded);
                return `
                        <div class="player-file-item ${!isPlayable ? 'disabled' : ''}"
                             data-index="${index}"
                             data-path="${file.path}"
                             data-transcoded-path="${file.transcoded_path || ''}"
                             data-transcode-status="${file.transcode_status || 0}"
                             data-streamable="${file.is_streamable}">
                            <span>${icon} ${file.path} ${transcodeStatusText}</span>
                            <span>${file.size_readable || formatSize(file.size)}</span>
                        </div>
                    `;
            }).join('')}
            `;

            // 绑定文件点击事件
            elements.playerFiles.querySelectorAll('.player-file-item:not(.disabled)').forEach(item => {
                item.addEventListener('click', () => {
                    const filePath = item.dataset.path;
                    const transcodedPath = item.dataset.transcodedPath;
                    const transcodeStatus = parseInt(item.dataset.transcodeStatus) || 0;
                    // Use transcoded file if available (status === 3 means completed)
                    const pathToPlay = (transcodeStatus === 3 && transcodedPath) ? transcodedPath : filePath;
                    playFile(pathToPlay, filePath);
                    // 更新选中状态
                    elements.playerFiles.querySelectorAll('.player-file-item').forEach(i => {
                        i.classList.remove('active');
                    });
                    item.classList.add('active');
                });
            });

            // 自动播放第一个可播放的文件
            // Playable = (streamable) OR (has transcoded version)
            const firstPlayable = data.files.find(f =>
                isVideoFile(f.path) && (f.is_streamable || (f.transcode_status === 3 && f.transcoded_path))
            );
            if (firstPlayable) {
                // Use transcoded file if available
                const pathToPlay = (firstPlayable.transcode_status === 3 && firstPlayable.transcoded_path)
                    ? firstPlayable.transcoded_path
                    : firstPlayable.path;
                playFile(pathToPlay, firstPlayable.path);
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

async function playFile(filePath, originalPath = null) {
    // originalPath is used for subtitle matching when playing transcoded files
    const subtitleMatchPath = originalPath || filePath;

    // Determine if this is a transcoded file
    let videoUrl;
    if (filePath.endsWith('_transcoded.mp4')) {
        // For transcoded files, use the transcoded endpoint
        // Remove leading ./ or / from the path
        let relativePath = filePath.replace(/^\.\/download\//, '').replace(/^\/download\//, '').replace(/^download\//, '');
        videoUrl = `${TORRENT_API}/transcoded/${encodeURIComponent(relativePath)}`;
    } else {
        // For original files, use the standard endpoint
        videoUrl = `${TORRENT_API}/file/${currentInfoHash}/${encodeURIComponent(filePath)}`;
    }

    // Clean up previous subtitle blob URL
    if (currentSubtitleBlobUrl) {
        URL.revokeObjectURL(currentSubtitleBlobUrl);
        currentSubtitleBlobUrl = null;
    }

    // Remove any existing track elements
    const existingTracks = elements.videoPlayer.querySelectorAll('track');
    existingTracks.forEach(track => track.remove());

    elements.videoPlayer.src = videoUrl;

    // Try to find and load matching subtitle automatically (use original path for matching)
    const matchingSubtitle = findMatchingSubtitle(subtitleMatchPath, currentSubtitleFiles);
    if (matchingSubtitle) {
        await loadAndAttachSubtitle(matchingSubtitle.path);
        // Update subtitle selector UI
        updateSubtitleSelection(matchingSubtitle.path);
    }

    elements.videoPlayer.play().catch(e => {
        console.log('Auto-play prevented:', e);
    });
}

// Load and attach subtitle to video
async function loadAndAttachSubtitle(subtitlePath) {
    if (!subtitlePath) {
        // Remove subtitle
        const existingTracks = elements.videoPlayer.querySelectorAll('track');
        existingTracks.forEach(track => track.remove());
        if (currentSubtitleBlobUrl) {
            URL.revokeObjectURL(currentSubtitleBlobUrl);
            currentSubtitleBlobUrl = null;
        }
        return;
    }

    const subtitleUrl = `${TORRENT_API}/file/${currentInfoHash}/${encodeURIComponent(subtitlePath)}`;
    const vttBlobUrl = await loadSubtitle(subtitleUrl, subtitlePath);

    if (vttBlobUrl) {
        // Remove any existing track elements
        const existingTracks = elements.videoPlayer.querySelectorAll('track');
        existingTracks.forEach(track => track.remove());

        // Clean up old blob URL
        if (currentSubtitleBlobUrl) {
            URL.revokeObjectURL(currentSubtitleBlobUrl);
        }
        currentSubtitleBlobUrl = vttBlobUrl;

        // Create and attach new track element
        const track = document.createElement('track');
        track.kind = 'subtitles';
        track.label = subtitlePath.split('/').pop();
        track.srclang = 'en';
        track.src = vttBlobUrl;
        track.default = true;

        elements.videoPlayer.appendChild(track);

        // Enable the track
        setTimeout(() => {
            if (elements.videoPlayer.textTracks.length > 0) {
                elements.videoPlayer.textTracks[0].mode = 'showing';
            }
        }, 100);

        showToast(`已加载字幕: ${subtitlePath.split('/').pop()}`, 'success');
    }
}

// Select subtitle from UI
function selectSubtitle(subtitlePath) {
    updateSubtitleSelection(subtitlePath);
    loadAndAttachSubtitle(subtitlePath);
}

// Update subtitle selector UI
function updateSubtitleSelection(selectedPath) {
    const subtitleOptions = document.querySelectorAll('.subtitle-option');
    subtitleOptions.forEach(option => {
        const optionPath = option.dataset.path;
        if (optionPath === selectedPath) {
            option.classList.add('active');
        } else {
            option.classList.remove('active');
        }
    });
}

// ============ 个人资料 ============

function loadProfile() {
    if (!currentUser) return;

    elements.profileEmail.textContent = currentUser.email;
    elements.profileNickname.textContent = currentUser.nickname;
    elements.profileAvatar.textContent = currentUser.nickname?.charAt(0).toUpperCase() || '👤';
}

// ============ 管理员功能 ============

function loadAdminPage() {
    // Initialize admin tabs
    initAdminTabs();
    // Load first tab by default
    loadAdminUsers();
    loadAdminStats();
}

function initAdminTabs() {
    const tabs = document.querySelectorAll('.admin-tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            // Update active content
            document.querySelectorAll('.admin-tab-content').forEach(c => c.classList.remove('active'));
            const targetContent = document.getElementById(`admin-tab-${tab.dataset.tab}`);
            if (targetContent) {
                targetContent.classList.add('active');
            }

            // Load content
            switch (tab.dataset.tab) {
                case 'users':
                    loadAdminUsers();
                    break;
                case 'resources':
                    loadAdminResources();
                    break;
                case 'stats':
                    loadAdminStats();
                    break;
            }
        });
    });

    // Search button events
    const searchUsersBtn = document.getElementById('admin-search-users-btn');
    if (searchUsersBtn) {
        searchUsersBtn.addEventListener('click', () => {
            adminUsersPage = 1;
            loadAdminUsers();
        });
    }

    const searchResourcesBtn = document.getElementById('admin-search-resources-btn');
    if (searchResourcesBtn) {
        searchResourcesBtn.addEventListener('click', () => {
            adminResourcesPage = 1;
            loadAdminResources();
        });
    }

    // Refresh stats button
    const refreshStatsBtn = document.getElementById('refresh-stats-btn');
    if (refreshStatsBtn) {
        refreshStatsBtn.addEventListener('click', loadAdminStats);
    }
}

async function loadAdminUsers() {
    const searchInput = document.getElementById('admin-user-search');
    const roleFilter = document.getElementById('admin-user-role-filter');
    const tbody = document.getElementById('admin-users-tbody');

    const params = new URLSearchParams({
        page: adminUsersPage,
        page_size: adminPageSize
    });

    if (searchInput && searchInput.value) {
        params.append('search', searchInput.value);
    }
    if (roleFilter && roleFilter.value) {
        params.append('role', roleFilter.value);
    }

    try {
        const data = await apiRequest(`${ADMIN_API}/users?${params}`);
        renderAdminUsers(data.users || [], data.total || 0);
    } catch (error) {
        console.error('加载用户列表失败:', error);
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center">加载失败</td></tr>';
        }
    }
}

function renderAdminUsers(users, total) {
    const tbody = document.getElementById('admin-users-tbody');
    if (!tbody) return;

    if (users.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center">暂无用户</td></tr>';
        return;
    }

    tbody.innerHTML = users.map(user => `
        <tr>
            <td>${user.id}</td>
            <td>${user.email}</td>
            <td>${user.nickname}</td>
            <td>
                <span class="role-badge ${user.is_super_admin ? 'super' : user.role}">${
                    user.is_super_admin ? '超级管理员' : (user.role === 'admin' ? '管理员' : '普通用户')
                }</span>
            </td>
            <td>${user.torrent_count || 0}</td>
            <td>${formatDate(user.created_at)}</td>
            <td class="actions">
                ${!user.is_super_admin ? `
                    <button class="btn btn-sm" onclick="toggleUserRole(${user.id}, '${user.role}')">
                        ${user.role === 'admin' ? '降级' : '升级'}
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="deleteUser(${user.id})">删除</button>
                ` : '<span style="color: var(--text-secondary)">-</span>'}
            </td>
        </tr>
    `).join('');

    // Render pagination
    renderAdminPagination('admin-users-pagination', total, adminUsersPage, (page) => {
        adminUsersPage = page;
        loadAdminUsers();
    });
}

async function loadAdminResources() {
    const searchInput = document.getElementById('admin-resource-search');
    const statusFilter = document.getElementById('admin-resource-status-filter');
    const tbody = document.getElementById('admin-resources-tbody');

    const params = new URLSearchParams({
        page: adminResourcesPage,
        page_size: adminPageSize
    });

    if (searchInput && searchInput.value) {
        params.append('search', searchInput.value);
    }
    if (statusFilter && statusFilter.value) {
        params.append('status', statusFilter.value);
    }

    try {
        const data = await apiRequest(`${ADMIN_API}/torrents?${params}`);
        renderAdminResources(data.torrents || [], data.total || 0);
    } catch (error) {
        console.error('加载资源列表失败:', error);
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center">加载失败</td></tr>';
        }
    }
}

function renderAdminResources(torrents, total) {
    const tbody = document.getElementById('admin-resources-tbody');
    if (!tbody) return;

    if (torrents.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align:center">暂无资源</td></tr>';
        return;
    }

    tbody.innerHTML = torrents.map(torrent => `
        <tr>
            <td title="${torrent.name}">${truncate(torrent.name, 30)}</td>
            <td>${formatSize(torrent.total_size)}</td>
            <td><span class="status-badge ${getStatusClass(torrent.status)}">${getStatusText(torrent.status)}</span></td>
            <td>${torrent.progress?.toFixed(1) || 0}%</td>
            <td><span class="transcode-badge ${getTranscodeClass(torrent.transcode_status)}">${getTranscodeText(torrent.transcode_status, torrent.status)}</span></td>
            <td>${torrent.creator_nickname || '-'}</td>
            <td>${formatDate(torrent.created_at)}</td>
            <td class="actions">
                <button class="btn btn-sm btn-danger" onclick="deleteAdminTorrent('${torrent.info_hash}')">删除</button>
            </td>
        </tr>
    `).join('');

    // Render pagination
    renderAdminPagination('admin-resources-pagination', total, adminResourcesPage, (page) => {
        adminResourcesPage = page;
        loadAdminResources();
    });
}

async function loadAdminStats() {
    try {
        const data = await apiRequest(`${ADMIN_API}/stats`);

        document.getElementById('stat-total-users').textContent = data.total_users || 0;
        document.getElementById('stat-total-torrents').textContent = data.total_torrents || 0;
        document.getElementById('stat-total-storage').textContent = formatSize(data.total_storage || 0);
        document.getElementById('stat-disk-usage').textContent = formatSize(data.actual_disk_usage || 0);
        document.getElementById('stat-active-downloads').textContent = data.active_downloads || 0;
        document.getElementById('stat-completed-downloads').textContent = data.completed_downloads || 0;
        document.getElementById('stat-transcoding-jobs').textContent = data.transcoding_jobs || 0;

        // System disk info
        if (data.disk_total) {
            document.getElementById('stat-disk-total').textContent = formatSize(data.disk_total);
            document.getElementById('stat-disk-free').textContent = formatSize(data.disk_free || 0);
        }
    } catch (error) {
        console.error('加载统计信息失败:', error);
    }
}

function renderAdminPagination(containerId, total, currentPage, onPageChange) {
    const container = document.getElementById(containerId);
    if (!container) return;

    const totalPages = Math.ceil(total / adminPageSize);
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    let html = '';

    // Previous button
    html += `<button ${currentPage === 1 ? 'disabled' : ''} onclick="window.adminPageChange('${containerId}', ${currentPage - 1})">上一页</button>`;

    // Page numbers
    for (let i = 1; i <= totalPages; i++) {
        if (i === 1 || i === totalPages || (i >= currentPage - 2 && i <= currentPage + 2)) {
            html += `<button class="${i === currentPage ? 'active' : ''}" onclick="window.adminPageChange('${containerId}', ${i})">${i}</button>`;
        } else if (i === currentPage - 3 || i === currentPage + 3) {
            html += '<button disabled>...</button>';
        }
    }

    // Next button
    html += `<button ${currentPage === totalPages ? 'disabled' : ''} onclick="window.adminPageChange('${containerId}', ${currentPage + 1})">下一页</button>`;

    container.innerHTML = html;

    // Store callback for global access
    window.adminPaginationCallbacks = window.adminPaginationCallbacks || {};
    window.adminPaginationCallbacks[containerId] = onPageChange;
}

// Global pagination handler
window.adminPageChange = function(containerId, page) {
    if (window.adminPaginationCallbacks && window.adminPaginationCallbacks[containerId]) {
        window.adminPaginationCallbacks[containerId](page);
    }
};

async function toggleUserRole(userId, currentRole) {
    const newRole = currentRole === 'admin' ? 'user' : 'admin';
    const action = newRole === 'admin' ? '升级为管理员' : '降级为普通用户';

    if (!confirm(`确定要将此用户${action}吗？`)) {
        return;
    }

    try {
        await apiRequest(`${ADMIN_API}/users/${userId}/role`, {
            method: 'PUT',
            body: JSON.stringify({ role: newRole })
        });
        showToast(`已${action}`, 'success');
        loadAdminUsers();
    } catch (error) {
        showToast(error.message || '操作失败', 'error');
    }
}

async function deleteUser(userId) {
    if (!confirm('确定要删除此用户吗？该用户的所有资源也将被删除！')) {
        return;
    }

    try {
        await apiRequest(`${ADMIN_API}/users/${userId}`, {
            method: 'DELETE'
        });
        showToast('用户已删除', 'success');
        loadAdminUsers();
        loadAdminStats();
    } catch (error) {
        showToast(error.message || '删除失败', 'error');
    }
}

async function deleteAdminTorrent(infoHash) {
    if (!confirm('确定要删除此资源吗？文件也将被删除！')) {
        return;
    }

    try {
        await apiRequest(`${ADMIN_API}/torrents/${infoHash}`, {
            method: 'DELETE'
        });
        showToast('资源已删除', 'success');
        loadAdminResources();
        loadAdminStats();
    } catch (error) {
        showToast(error.message || '删除失败', 'error');
    }
}

// Helper functions for admin
function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function truncate(str, maxLen) {
    if (!str) return '';
    return str.length > maxLen ? str.substring(0, maxLen) + '...' : str;
}

function getStatusClass(status) {
    const classMap = {
        0: 'pending',
        1: 'downloading',
        2: 'completed',
        3: 'failed',
        4: 'paused'
    };
    return classMap[status] || 'pending';
}

function getTranscodeText(status, downloadStatus = 2) {
    // If download is not completed and transcode_status is 0, show "待检测"
    if (status === 0 && downloadStatus !== 2) {
        return '待检测';
    }
    const textMap = {
        0: '无需转码',
        1: '待转码',
        2: '转码中',
        3: '已完成',
        4: '失败'
    };
    return textMap[status] || '未知';
}

function getTranscodeClass(status) {
    const classMap = {
        0: 'none',
        1: 'pending',
        2: 'processing',
        3: 'completed',
        4: 'failed'
    };
    return classMap[status] || 'none';
}

// Get transcode status text for individual file in player
function getFileTranscodeStatus(file) {
    if (!file.transcode_status || file.transcode_status === 0) {
        return '';
    }
    const statusMap = {
        1: '<span class="file-transcode-badge pending">待转码</span>',
        2: '<span class="file-transcode-badge processing">转码中</span>',
        3: '<span class="file-transcode-badge completed">已转码</span>',
        4: '<span class="file-transcode-badge failed">转码失败</span>'
    };
    return statusMap[file.transcode_status] || '';
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
window.selectSubtitle = selectSubtitle;

// Admin functions
window.toggleUserRole = toggleUserRole;
window.deleteUser = deleteUser;
window.deleteAdminTorrent = deleteAdminTorrent;
