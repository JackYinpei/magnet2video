// API 基础配置
const API_BASE = '/api/v1/torrent';

// 状态
let currentInfoHash = null;
let parsedTorrent = null;
let progressInterval = null;

// DOM 元素
const elements = {
    // 页面
    pageLibrary: document.getElementById('page-library'),
    pageAdd: document.getElementById('page-add'),
    pageDownloads: document.getElementById('page-downloads'),
    pagePlayer: document.getElementById('page-player'),

    // 导航
    navLinks: document.querySelectorAll('.nav-link'),

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

    // 下载
    downloadsList: document.getElementById('downloads-list'),

    // 播放器
    backBtn: document.getElementById('back-btn'),
    playerTitle: document.getElementById('player-title'),
    videoPlayer: document.getElementById('video-player'),
    playerFiles: document.getElementById('player-files'),

    // 通用
    toast: document.getElementById('toast'),
    loading: document.getElementById('loading')
};

// 工具函数
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

// API 请求
async function apiRequest(endpoint, options = {}) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });
        const data = await response.json();
        if (data.error) {
            throw new Error(data.error.message || '请求失败');
        }
        return data.data;
    } catch (error) {
        throw error;
    }
}

// 页面导航
function navigateTo(pageName) {
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
    if (pageName === 'library') {
        loadLibrary();
    } else if (pageName === 'downloads') {
        loadDownloads();
        startProgressPolling();
    } else {
        stopProgressPolling();
    }
}

// 解析磁力链接
async function parseMagnet() {
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
        const data = await apiRequest('/parse', {
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

// 渲染文件列表
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

// 获取选中的文件索引
function getSelectedFiles() {
    const checkboxes = elements.fileList.querySelectorAll('input[type="checkbox"]:checked');
    return Array.from(checkboxes).map(cb => parseInt(cb.dataset.index));
}

// 选择操作
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

// 开始下载
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
        await apiRequest('/download', {
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

// 重置添加表单
function resetAddForm() {
    elements.magnetInput.value = '';
    elements.trackerInput.value = '';
    elements.fileSelection.classList.add('hidden');
    elements.fileList.innerHTML = '';
    parsedTorrent = null;
    currentInfoHash = null;
}

// 加载媒体库
async function loadLibrary() {
    try {
        const data = await apiRequest('/list');
        renderLibrary(data.torrents || []);
    } catch (error) {
        console.error('加载媒体库失败:', error);
    }
}

// 渲染媒体库
function renderLibrary(torrents) {
    if (torrents.length === 0) {
        elements.libraryGrid.innerHTML = `
            <div class="empty-state">
                <p>暂无媒体，点击"添加"开始下载</p>
            </div>
        `;
        return;
    }

    elements.libraryGrid.innerHTML = torrents.map(torrent => `
        <div class="poster-card" data-infohash="${torrent.info_hash}">
            <div class="poster-image">
                ${torrent.poster_path
            ? `<img src="${torrent.poster_path}" alt="${torrent.name}">`
            : '🎬'}
            </div>
            <div class="poster-info">
                <div class="poster-title" title="${torrent.name}">${torrent.name}</div>
                <div class="poster-meta">${formatSize(torrent.total_size)} · ${getStatusText(torrent.status)}</div>
                ${torrent.status !== 2 ? `
                    <div class="poster-progress">
                        <div class="poster-progress-bar" style="width: ${torrent.progress}%"></div>
                    </div>
                ` : ''}
            </div>
        </div>
    `).join('');

    // 绑定点击事件
    elements.libraryGrid.querySelectorAll('.poster-card').forEach(card => {
        card.addEventListener('click', () => {
            openPlayer(card.dataset.infohash);
        });
    });
}

// 加载下载列表
async function loadDownloads() {
    try {
        const data = await apiRequest('/list');
        renderDownloads(data.torrents || []);
    } catch (error) {
        console.error('加载下载列表失败:', error);
    }
}

// 渲染下载列表
function renderDownloads(torrents) {
    // 过滤出非完成的任务
    const activeTorrents = torrents.filter(t => t.status !== 2);

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

// 暂停下载
async function pauseDownload(infoHash) {
    try {
        await apiRequest('/pause', {
            method: 'POST',
            body: JSON.stringify({ info_hash: infoHash })
        });
        showToast('已暂停', 'success');
        loadDownloads();
    } catch (error) {
        showToast(error.message || '暂停失败', 'error');
    }
}

// 继续下载
async function resumeDownload(infoHash) {
    try {
        await apiRequest('/resume', {
            method: 'POST',
            body: JSON.stringify({
                info_hash: infoHash,
                selected_files: [] // 恢复所有已选文件
            })
        });
        showToast('已继续', 'success');
        loadDownloads();
    } catch (error) {
        showToast(error.message || '继续失败', 'error');
    }
}

// 删除种子
async function removeTorrent(infoHash) {
    if (!confirm('确定要删除这个任务吗？')) {
        return;
    }

    try {
        await apiRequest('/remove', {
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

// 开始进度轮询
function startProgressPolling() {
    stopProgressPolling();
    progressInterval = setInterval(() => {
        loadDownloads();
    }, 3000);
}

// 停止进度轮询
function stopProgressPolling() {
    if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
    }
}

// 打开播放器
async function openPlayer(infoHash) {
    showLoading();

    try {
        const data = await apiRequest(`/detail/${infoHash}`);

        currentInfoHash = infoHash;
        elements.playerTitle.textContent = data.name;

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

// 播放文件
function playFile(filePath) {
    const videoUrl = `${API_BASE}/file/${currentInfoHash}/${encodeURIComponent(filePath)}`;
    elements.videoPlayer.src = videoUrl;
    elements.videoPlayer.play().catch(e => {
        console.log('Auto-play prevented:', e);
    });
}

// 初始化事件监听
function initEventListeners() {
    // 导航
    elements.navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            navigateTo(link.dataset.page);
        });
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
        navigateTo('library');
    });

    // 键盘快捷键
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && elements.pagePlayer.classList.contains('active')) {
            elements.videoPlayer.pause();
            elements.videoPlayer.src = '';
            navigateTo('library');
        }
    });
}

// 初始化
function init() {
    initEventListeners();
    loadLibrary();
}

// 启动应用
document.addEventListener('DOMContentLoaded', init);
