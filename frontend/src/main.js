import "./style.css";

import {
  AddBookmark,
  DeleteBookmark,
  GoToBookmark,
  JumpToOffset,
  JumpToPercent,
  ListBookmarks,
  ListFolderFiles,
  OpenFile,
  ReadAroundOffset,
  ReadNextPage,
  ReadPreviousPage,
  SaveProgress,
  SearchForward,
  SearchHitPage,
  SearchHitPageByIndex,
  SearchHitPreviews,
  SelectFile,
  SelectFolder,
  SetLanguage,
  StartSearch,
} from "../wailsjs/go/main/App";
import { EventsOn, OnFileDrop } from "../wailsjs/runtime/runtime";

const LOAD_THRESHOLD_PX = 600;
const PREFETCH_THRESHOLD_PX = 1400;
const LEGACY_LANG_STORAGE_KEY = "bf-reader.language";
const LEGACY_FONT_SIZE_STORAGE_KEY = "bf-reader.fontSize";
const LANG_STORAGE_KEY = "bigtext-reader.language";
const FONT_SIZE_STORAGE_KEY = "bigtext-reader.fontSize";
const DEFAULT_PAGE_SIZE = 60;

const messages = {
  zh: {
    "menu.file": "文件",
    "menu.open": "打开文件...",
    "menu.openFolder": "打开文件夹...",
    "menu.reload": "重新加载",
    "menu.about": "关于",
    "file.noFile": "尚未打开文件",
    "file.pathPlaceholder": "选择一个 TXT / LOG / 文本文件",
    "tabs.files": "文件",
    "tabs.bookmarks": "书签",
    "files.empty": "暂无文件",
    "files.openFile": "打开文件...",
    "files.openFolder": "打开文件夹...",
    "bookmark.placeholder": "书签名",
    "bookmark.save": "保存书签",
    "bookmark.count": "书签 {count}",
    "bookmark.empty": "暂无书签",
    "bookmark.delete": "删除",
    "bookmark.deleteTitle": "删除书签",
    "toolbar.jump": "定位",
    "toolbar.prev": "上段",
    "toolbar.next": "下段",
    "toolbar.encoding": "编码",
    "toolbar.fontSize": "字号",
    "search.placeholder": "搜索关键字，回车查找",
    "search.find": "查找",
    "search.prev": "上一个",
    "search.next": "下一个",
    "search.none": "未搜索",
    "search.title": "搜索结果",
    "search.close": "× 关闭",
    "search.scanning": "正在扫描文件",
    "search.counting": "正在统计...",
    "search.prompt": "输入关键词后显示结果",
    "search.noMatch": "没有找到匹配项",
    "search.noResult": "暂无结果",
    "search.total": "共 {total} 个",
    "search.current": "第 {current} / 共 {total} 个",
    "search.line": "行",
    "search.loading": "加载中...",
    "empty.title": "打开一个大文本文件",
    "empty.desc": "支持 GB 级 TXT / LOG，滚动时自动加载上下文。",
    "status.ready": "就绪",
    "status.about": "BigText Reader · GB 级大文本阅读器",
    "about.title": "BigText Reader",
    "about.tagline": "GB 级大文本阅读器",
    "about.description":
      "面向超大 TXT / LOG 文件设计，按需读取内容，支持 UTF-8、GBK、搜索、书签和阅读进度保存。",
    "about.versionLabel": "版本",
    "about.versionValue": "本地构建",
    "about.stackLabel": "技术栈",
    "about.stackValue": "Go / Wails / Vanilla JS",
    "about.encodingLabel": "编码支持",
    "about.encodingValue": "UTF-8 / GBK / 自动检测",
    "about.authorLabel": "作者",
    "about.authorValue": "weiwt",
    "about.emailLabel": "邮箱",
    "about.emailValue": "taoao.wei@gmail.com",
    "about.repositoryLabel": "仓库",
    "about.repositoryValue": "github.com/weiwentao996/bigtext-reader",
    "about.close": "关闭",
    "status.opening": "正在打开文件...",
    "status.reencoding": "正在切换编码...",
    "status.encodingChanged": "编码已切换为 {encoding}",
    "status.opened": "文件已打开",
    "status.resumed": "已恢复上次进度",
    "status.jumping": "正在跳转...",
    "status.jumpedOffset": "已定位到 offset {offset}",
    "status.found": "找到 {total} 个匹配项",
    "status.searching": "正在统计搜索结果...",
    "status.jumpedHit": "已跳转到第 {index} 个匹配项",
    "status.bookmarkSaved": "书签已保存",
    "status.bookmarkJumped": "已跳转到书签",
    "status.bookmarkDeleted": "书签已删除",
    "state.idle": "空闲",
    "state.reading": "阅读中",
    "state.truncated": "已截断",
    "error.selectFile": "请先选择文件",
  },
  en: {
    "menu.file": "File",
    "menu.open": "Open...",
    "menu.openFolder": "Open Folder...",
    "menu.reload": "Reload",
    "menu.about": "About",
    "file.noFile": "No file open",
    "file.pathPlaceholder": "Select a TXT / LOG / text file",
    "tabs.files": "Files",
    "tabs.bookmarks": "Bookmarks",
    "files.empty": "No files available",
    "files.openFile": "Open File...",
    "files.openFolder": "Open Folder...",
    "bookmark.placeholder": "Bookmark name",
    "bookmark.save": "Save",
    "bookmark.count": "Bookmarks {count}",
    "bookmark.empty": "No bookmarks",
    "bookmark.delete": "Delete",
    "bookmark.deleteTitle": "Delete bookmark",
    "toolbar.jump": "Go",
    "toolbar.prev": "Previous",
    "toolbar.next": "Next",
    "toolbar.encoding": "Encoding",
    "toolbar.fontSize": "Font",
    "search.placeholder": "Search keyword, press Enter",
    "search.find": "Find",
    "search.prev": "Previous",
    "search.next": "Next",
    "search.none": "Not searched",
    "search.title": "Search results",
    "search.close": "× Close",
    "search.scanning": "Scanning file",
    "search.counting": "Counting...",
    "search.prompt": "Enter a keyword to show results",
    "search.noMatch": "No matches found",
    "search.noResult": "No results",
    "search.total": "{total} total",
    "search.current": "{current} of {total}",
    "search.line": "line",
    "search.loading": "loading...",
    "empty.title": "Open a large text file",
    "empty.desc":
      "Supports GB-scale TXT / LOG files and loads context while scrolling.",
    "status.ready": "Ready",
    "status.about": "BigText Reader · GB-scale text reader",
    "about.title": "BigText Reader",
    "about.tagline": "GB-scale text reader",
    "about.description":
      "Built for very large TXT / LOG files. It reads on demand and supports UTF-8, GBK, search, bookmarks, and reading progress.",
    "about.versionLabel": "Version",
    "about.versionValue": "Local build",
    "about.stackLabel": "Stack",
    "about.stackValue": "Go / Wails / Vanilla JS",
    "about.encodingLabel": "Encoding",
    "about.encodingValue": "UTF-8 / GBK / auto detect",
    "about.authorLabel": "Author",
    "about.authorValue": "weiwt",
    "about.emailLabel": "Email",
    "about.emailValue": "taoao.wei@gmail.com",
    "about.repositoryLabel": "Repository",
    "about.repositoryValue": "github.com/weiwentao996/bigtext-reader",
    "about.close": "Close",
    "status.opening": "Opening file...",
    "status.reencoding": "Switching encoding...",
    "status.encodingChanged": "Encoding switched to {encoding}",
    "status.opened": "File opened",
    "status.resumed": "Resumed previous position",
    "status.jumping": "Jumping...",
    "status.jumpedOffset": "Jumped to offset {offset}",
    "status.found": "Found {total} matches",
    "status.searching": "Counting search results...",
    "status.jumpedHit": "Jumped to match {index}",
    "status.bookmarkSaved": "Bookmark saved",
    "status.bookmarkJumped": "Jumped to bookmark",
    "status.bookmarkDeleted": "Bookmark deleted",
    "state.idle": "idle",
    "state.reading": "reading",
    "state.truncated": "truncated",
    "error.selectFile": "Select a file first",
  },
};

function normalizeLanguage(lang) {
  return String(lang).toLowerCase().startsWith("en") ? "en" : "zh";
}

function loadLanguage() {
  const stored =
    localStorage.getItem(LANG_STORAGE_KEY) ??
    localStorage.getItem(LEGACY_LANG_STORAGE_KEY);
  if (stored && !localStorage.getItem(LANG_STORAGE_KEY)) {
    localStorage.setItem(LANG_STORAGE_KEY, normalizeLanguage(stored));
  }
  return normalizeLanguage(stored || navigator.language || "zh");
}

function loadFontSize() {
  const stored =
    localStorage.getItem(FONT_SIZE_STORAGE_KEY) ??
    localStorage.getItem(LEGACY_FONT_SIZE_STORAGE_KEY);
  const value = clampFontSize(Number(stored) || 16.5);
  if (stored && !localStorage.getItem(FONT_SIZE_STORAGE_KEY)) {
    localStorage.setItem(FONT_SIZE_STORAGE_KEY, String(value));
  }
  return value;
}

function clampFontSize(value) {
  if (!Number.isFinite(value)) return 16.5;
  return Math.min(32, Math.max(12, value));
}

function t(key, params = {}) {
  const template =
    messages[state?.language || loadLanguage()]?.[key] ??
    messages.zh[key] ??
    key;
  return template.replace(/\{(\w+)\}/g, (_, name) => params[name] ?? "");
}

const app = document.querySelector("#app");

app.innerHTML = `
  <div class="app-shell typora-shell">
    <header class="topbar">
      <nav class="menu-strip" aria-label="Application menu">
        <div class="menu-root">
          <button id="fileMenuButton" class="menu-item" type="button" data-i18n="menu.file">File</button>
          <div id="fileMenu" class="menu-dropdown">
            <button id="menuOpenFile" class="menu-command" type="button"><span data-i18n="menu.open">Open...</span><kbd>Ctrl+O</kbd></button>
            <button id="menuOpenFolder" class="menu-command" type="button"><span data-i18n="menu.openFolder">Open Folder...</span></button>
            <div class="menu-separator"></div>
            <button id="menuReload" class="menu-command" type="button"><span data-i18n="menu.reload">Reload</span></button>
          </div>
        </div>
        <button id="aboutButton" class="menu-item" type="button" data-i18n="menu.about">关于</button>
      </nav>

      <div class="file-info">
        <div id="fileTitle" class="file-title" data-i18n="file.noFile">尚未打开文件</div>
        <input id="filePath" class="file-path" data-i18n-placeholder="file.pathPlaceholder" placeholder="选择一个 TXT / LOG / 文本文件" readonly />
      </div>

      <label class="language-control">
        <select id="languageSelect" aria-label="Language">
          <option value="zh">中文</option>
          <option value="en">English</option>
        </select>
      </label>
    </header>

    <main class="workspace">
      <aside class="side-panel left-sidebar">
        <div class="side-tabs">
          <button id="filesTab" class="side-tab active" type="button" data-i18n="tabs.files">文件</button>
          <button id="bookmarksTab" class="side-tab" type="button" data-i18n="tabs.bookmarks">书签</button>
        </div>

        <section id="filesPanel" class="side-section active">
          <div id="folderFiles" class="folder-files">
            <div class="empty-folder-files" data-i18n="files.empty">No Files Available</div>
          </div>
          <div class="file-actions">
            <button id="openFileButton" class="open-file-button" type="button" data-i18n="files.openFile">Open File...</button>
            <button id="openFolderButton" class="open-folder-button" type="button" data-i18n="files.openFolder">Open Folder...</button>
          </div>
        </section>

        <section id="bookmarksPanel" class="side-section bookmarks-section">
          <div id="bookmarks" class="bookmarks"></div>
          <section class="sidebar-bookmark-editor">
            <input id="bookmarkName" class="compact-input" data-i18n-placeholder="bookmark.placeholder" placeholder="书签名" />
            <button id="bookmarkButton" class="button subtle" data-i18n="bookmark.save">保存书签</button>
          </section>
        </section>
      </aside>

      <section class="reader-stage">
        <section class="commandbar reader-toolbar">
          <div class="command-group">
            <input id="jumpInput" class="compact-input" placeholder="50% / offset" />
            <button id="jumpButton" class="button subtle" data-i18n="toolbar.jump">定位</button>
          </div>

          <div class="segmented-actions">
            <button id="prevPage" class="button icon-button" data-i18n="toolbar.prev">上段</button>
            <button id="nextPage" class="button icon-button" data-i18n="toolbar.next">下段</button>
          </div>

          <label class="mini-field"><span data-i18n="toolbar.encoding">编码</span>
            <select id="encoding">
              <option value="auto">auto</option>
              <option value="utf8">utf8</option>
              <option value="gbk">gbk</option>
            </select>
          </label>

          <label class="mini-field"><span data-i18n="toolbar.fontSize">字号</span>
            <input id="fontSize" type="number" min="12" max="32" step="0.5" value="16.5" />
          </label>
        </section>

        <section class="searchbar">
          <input id="searchInput" class="search-input" data-i18n-placeholder="search.placeholder" placeholder="搜索关键字，回车查找" />
          <button id="searchButton" class="button subtle" data-i18n="search.find">查找</button>
          <button id="prevSearch" class="button subtle" data-i18n="search.prev">上一个</button>
          <button id="nextSearch" class="button subtle" data-i18n="search.next">下一个</button>
          <span id="searchSummary" class="search-summary" data-i18n="search.none">未搜索</span>
        </section>

        <section id="searchPanel" class="search-panel">
          <div class="search-panel-header">
            <span data-i18n="search.title">搜索结果</span>
            <button id="closeSearchPanel" class="close-search-panel" type="button" data-i18n="search.close">× 关闭</button>
          </div>
          <div id="searchResults" class="search-results"></div>
        </section>

        <div class="reader-surface">
          <div id="reader" class="reader empty">
            <div id="pageContainer" class="page-container">
              <div class="empty-state">
                <div class="empty-icon">▤</div>
                <div class="empty-title" data-i18n="empty.title">打开一个大文本文件</div>
                <div class="empty-desc" data-i18n="empty.desc">支持 GB 级 TXT / LOG，滚动时自动加载上下文。</div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="statusbar">
      <div id="statusText" class="status-message" data-i18n="status.ready">就绪</div>
      <div class="progress-wrap">
        <div class="progress-rail"><div id="progressBar" class="progress-bar"></div></div>
        <span id="progressText" class="progress-label">0.0000%</span>
      </div>
      <div class="meta-chips">
        <span id="offsetChip" class="chip">offset 0</span>
        <span id="encodingChip" class="chip">encoding -</span>
        <span id="stateChip" class="chip muted-chip">idle</span>
      </div>
    </footer>

    <div id="aboutOverlay" class="about-overlay" aria-hidden="true">
      <section class="about-dialog" role="dialog" aria-modal="true" aria-labelledby="aboutTitle">
        <button id="aboutCloseIcon" class="about-close-icon" type="button" aria-label="Close">×</button>
        <div class="about-mark">BF</div>
        <h2 id="aboutTitle" class="about-title" data-i18n="about.title">BigText Reader</h2>
        <div class="about-tagline" data-i18n="about.tagline">GB 级大文本阅读器</div>
        <p class="about-description" data-i18n="about.description">面向超大 TXT / LOG 文件设计，按需读取内容，支持 UTF-8、GBK、搜索、书签和阅读进度保存。</p>
        <dl class="about-meta">
          <div><dt data-i18n="about.versionLabel">版本</dt><dd data-i18n="about.versionValue">本地构建</dd></div>
          <div><dt data-i18n="about.stackLabel">技术栈</dt><dd data-i18n="about.stackValue">Go / Wails / Vanilla JS</dd></div>
          <div><dt data-i18n="about.encodingLabel">编码支持</dt><dd data-i18n="about.encodingValue">UTF-8 / GBK / 自动检测</dd></div>
          <div><dt data-i18n="about.authorLabel">作者</dt><dd data-i18n="about.authorValue">weiwt</dd></div>
          <div><dt data-i18n="about.emailLabel">邮箱</dt><dd data-i18n="about.emailValue">taoao.wei@gmail.com</dd></div>
          <div><dt data-i18n="about.repositoryLabel">仓库</dt><dd data-i18n="about.repositoryValue">github.com/weiwentao996/bigtext-reader</dd></div>
        </dl>
        <div class="about-actions">
          <button id="aboutCloseButton" class="button primary" type="button" data-i18n="about.close">关闭</button>
        </div>
      </section>
    </div>
  </div>
`;

const state = {
  language: loadLanguage(),
  path: "",
  meta: null,
  bookmarks: [],
  folderPath: "",
  folderFiles: [],
  pages: new Map(),
  order: [],
  pending: new Set(),
  maxMountedPages: 9,
  loadingPrev: false,
  loadingNext: false,
  suppressScroll: false,
  saveTimer: null,
  fontSize: loadFontSize(),
  searchMatch: null,
  lastSearchKeyword: "",
  lastSearchHitOffset: -1,
  lastSearchHitByteLength: 0,
  searchSeq: 0,
  searching: false,
  searchSessionId: "",
  searchPreviewCache: new Map(),
  searchPreviewPending: new Set(),
  searchTotal: 0,
  searchCurrentIndex: -1,
  searchResultsKeyword: "",
  searchStatsLoading: false,
  searchPanelVisible: false,
  searchVirtual: {
    rowHeight: 42,
    pageSize: 100,
    overscan: 8,
    maxCachedRows: 2200,
  },
  sidePanelTab: "files",
};

const el = {
  fileMenuButton: document.querySelector("#fileMenuButton"),
  fileMenu: document.querySelector("#fileMenu"),
  menuOpenFile: document.querySelector("#menuOpenFile"),
  menuOpenFolder: document.querySelector("#menuOpenFolder"),
  menuReload: document.querySelector("#menuReload"),
  aboutButton: document.querySelector("#aboutButton"),
  aboutOverlay: document.querySelector("#aboutOverlay"),
  aboutCloseIcon: document.querySelector("#aboutCloseIcon"),
  aboutCloseButton: document.querySelector("#aboutCloseButton"),
  languageSelect: document.querySelector("#languageSelect"),
  openFileButton: document.querySelector("#openFileButton"),
  openFolderButton: document.querySelector("#openFolderButton"),
  folderFiles: document.querySelector("#folderFiles"),
  filePath: document.querySelector("#filePath"),
  encoding: document.querySelector("#encoding"),
  fontSize: document.querySelector("#fontSize"),
  prevPage: document.querySelector("#prevPage"),
  nextPage: document.querySelector("#nextPage"),
  jumpInput: document.querySelector("#jumpInput"),
  jumpButton: document.querySelector("#jumpButton"),
  searchInput: document.querySelector("#searchInput"),
  searchButton: document.querySelector("#searchButton"),
  bookmarkName: document.querySelector("#bookmarkName"),
  bookmarkButton: document.querySelector("#bookmarkButton"),
  reader: document.querySelector("#reader"),
  pageContainer: document.querySelector("#pageContainer"),
  bookmarks: document.querySelector("#bookmarks"),
  filesTab: document.querySelector("#filesTab"),
  bookmarksTab: document.querySelector("#bookmarksTab"),
  filesPanel: document.querySelector("#filesPanel"),
  bookmarksPanel: document.querySelector("#bookmarksPanel"),
  searchPanel: document.querySelector("#searchPanel"),
  closeSearchPanel: document.querySelector("#closeSearchPanel"),
  prevSearch: document.querySelector("#prevSearch"),
  nextSearch: document.querySelector("#nextSearch"),
  searchSummary: document.querySelector("#searchSummary"),
  searchResults: document.querySelector("#searchResults"),
  statusText: document.querySelector("#statusText"),
  progressText: document.querySelector("#progressText"),
  fileTitle: document.querySelector("#fileTitle"),
  progressBar: document.querySelector("#progressBar"),
  offsetChip: document.querySelector("#offsetChip"),
  encodingChip: document.querySelector("#encodingChip"),
  stateChip: document.querySelector("#stateChip"),
};

el.fileMenuButton.addEventListener("click", (event) => {
  event.stopPropagation();
  toggleFileMenu();
});
el.menuOpenFile.addEventListener("click", () => run(selectAndOpenFile));
el.menuOpenFolder.addEventListener("click", () => run(selectAndOpenFolder));
el.menuReload.addEventListener("click", () => run(openCurrentFile));
el.aboutButton.addEventListener("click", showAboutDialog);
el.aboutCloseIcon.addEventListener("click", hideAboutDialog);
el.aboutCloseButton.addEventListener("click", hideAboutDialog);
el.aboutOverlay.addEventListener("click", (event) => {
  if (event.target === el.aboutOverlay) hideAboutDialog();
});
el.languageSelect.addEventListener("change", () => run(changeLanguage));
el.openFileButton.addEventListener("click", () => run(selectAndOpenFile));
el.openFolderButton.addEventListener("click", () => run(selectAndOpenFolder));
el.encoding.addEventListener("change", () => run(changeEncoding));
el.fontSize.addEventListener("change", changeFontSize);
el.fontSize.addEventListener("input", changeFontSize);
el.nextPage.addEventListener("click", () => run(loadNext));
el.prevPage.addEventListener("click", () => run(loadPrevious));
el.jumpButton.addEventListener("click", () => run(jump));
el.searchButton.addEventListener("click", () => run(search));
el.prevSearch.addEventListener("click", () =>
  run(() => goRelativeSearchHit(-1)),
);
el.nextSearch.addEventListener("click", () =>
  run(() => goRelativeSearchHit(1)),
);
el.closeSearchPanel.addEventListener("click", () => hideSearchResults());
el.bookmarkButton.addEventListener("click", () => run(addBookmark));
el.filesTab.addEventListener("click", () => setSidePanelTab("files"));
el.bookmarksTab.addEventListener("click", () => setSidePanelTab("bookmarks"));
el.reader.addEventListener("scroll", onReaderScroll);
el.searchResults.addEventListener("scroll", () => {
  renderSearchResults();
});
el.searchInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter") run(search);
});
el.jumpInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter") run(jump);
});

window.addEventListener("click", () => closeFileMenu());
el.languageSelect.value = state.language;
el.fontSize.value = String(state.fontSize);
applyFontSize();
applyLocale();
run(() => SetLanguage(state.language));
EventsOn("app:open-file", (path) => run(() => openFilePath(path)));
OnFileDrop((_x, _y, paths) => run(() => openFilePath(paths?.[0])), false);

window.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && el.aboutOverlay.classList.contains("open")) {
    hideAboutDialog();
    return;
  }
  if (event.ctrlKey && event.key.toLowerCase() === "o") {
    event.preventDefault();
    run(selectAndOpenFile);
    return;
  }
  if (
    event.target instanceof HTMLInputElement ||
    event.target instanceof HTMLSelectElement
  )
    return;
  if (event.key === "ArrowRight" || event.key.toLowerCase() === "n")
    run(loadNext);
  if (event.key === "ArrowLeft" || event.key.toLowerCase() === "p")
    run(loadPrevious);
});

async function changeLanguage() {
  state.language = normalizeLanguage(el.languageSelect.value);
  el.languageSelect.value = state.language;
  localStorage.setItem(LANG_STORAGE_KEY, state.language);
  await SetLanguage(state.language);
  applyLocale();
  renderBookmarks();
  renderFolderFiles();
  renderSearchResults();
  updateVisibleProgress();
}

function applyLocale() {
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    if (node === el.fileTitle && state.meta) return;
    node.textContent = t(node.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  });
  document.querySelectorAll("[data-i18n-title]").forEach((node) => {
    node.title = t(node.dataset.i18nTitle);
  });
  if (!state.meta) {
    el.fileTitle.textContent = t("file.noFile");
  }
}

async function selectAndOpenFile() {
  closeFileMenu();
  const path = await SelectFile();
  if (path) {
    await openFilePath(path);
  }
}

async function openFilePath(path) {
  if (!path) return;
  state.path = path;
  el.filePath.value = path;
  await openCurrentFile();
}

async function selectAndOpenFolder() {
  closeFileMenu();
  const folder = await SelectFolder();
  if (!folder) return;
  state.folderPath = folder;
  state.folderFiles = await ListFolderFiles(folder);
  renderFolderFiles();
}

function toggleFileMenu() {
  el.fileMenu.classList.toggle("open");
}

function closeFileMenu() {
  el.fileMenu.classList.remove("open");
}

function showAboutDialog() {
  closeFileMenu();
  el.aboutOverlay.classList.add("open");
  el.aboutOverlay.setAttribute("aria-hidden", "false");
  el.aboutCloseButton.focus();
  setStatus(t("status.about"));
}

function hideAboutDialog() {
  el.aboutOverlay.classList.remove("open");
  el.aboutOverlay.setAttribute("aria-hidden", "true");
}

function changeFontSize() {
  state.fontSize = clampFontSize(Number(el.fontSize.value));
  el.fontSize.value = String(state.fontSize);
  localStorage.setItem(FONT_SIZE_STORAGE_KEY, String(state.fontSize));
  applyFontSize();
}

function applyFontSize() {
  document.documentElement.style.setProperty(
    "--reader-font-size",
    `${state.fontSize}px`,
  );
}

async function changeEncoding() {
  if (!state.meta) return;
  const offset = getVisiblePageOffset();
  await SaveProgress(offset);
  await openCurrentFile({
    encoding: el.encoding.value,
    busyText: t("status.reencoding"),
  });
  const page = await JumpToOffset(offset);
  await showAnchorPage(page);
  setStatus(t("status.encodingChanged", { encoding: el.encoding.value }));
}

async function openCurrentFile(options = {}) {
  closeFileMenu();
  const path = state.path || el.filePath.value;
  if (!path) throw new Error(t("error.selectFile"));
  setBusy(options.busyText || t("status.opening"));
  const result = await OpenFile(
    path,
    options.encoding || el.encoding.value,
    DEFAULT_PAGE_SIZE,
  );
  state.path = result.meta.path || path;
  el.filePath.value = state.path;
  state.meta = result.meta;
  state.bookmarks = result.bookmarks || [];
  resetSearchStateForFileChange();
  syncCurrentFileToList(result.meta);
  el.encoding.value = result.meta.encoding;
  el.fileTitle.textContent = result.meta.name || t("status.opened");
  await showAnchorPage(result.page);
  renderBookmarks();
  renderSearchResults();
  renderFolderFiles();
  setSidePanelTab(state.sidePanelTab);
  setStatus(result.resumed ? t("status.resumed") : t("status.opened"));
}

async function showAnchorPage(page, searchMatch = null) {
  const windowResult = await ReadAroundOffset(page.startOffset, 1, 2);
  resetReaderState();
  state.searchMatch = searchMatch;
  const pages =
    windowResult.pages && windowResult.pages.length
      ? windowResult.pages
      : [page];
  pages.forEach(mountPage);
  el.reader.classList.remove("empty");
  await nextFrame();
  scrollPageToTop(windowResult.anchor?.startOffset ?? page.startOffset);
  await SaveProgress(windowResult.anchor?.startOffset ?? page.startOffset);
  updateVisibleProgress();
}

function resetReaderState() {
  state.pages.clear();
  state.order = [];
  state.pending.clear();
  state.loadingPrev = false;
  state.loadingNext = false;
  el.pageContainer.innerHTML = "";
  el.reader.scrollTop = 0;
}

function mountPage(page) {
  if (!page || state.pages.has(page.startOffset)) return;

  state.pages.set(page.startOffset, page);
  state.order.push(page.startOffset);
  state.order.sort((a, b) => a - b);

  const pageEl = document.createElement("pre");
  pageEl.className = "page";
  pageEl.dataset.start = String(page.startOffset);
  pageEl.dataset.end = String(page.endOffset);
  renderPageText(pageEl, page);

  const nextOffset = state.order.find((offset) => offset > page.startOffset);
  if (nextOffset !== undefined) {
    const nextEl = getPageElement(nextOffset);
    el.pageContainer.insertBefore(pageEl, nextEl);
  } else {
    el.pageContainer.appendChild(pageEl);
  }
}

function removePage(offset, preserveScroll) {
  const pageEl = getPageElement(offset);
  if (!pageEl) return;
  const height = pageEl.offsetHeight;
  pageEl.remove();
  state.pages.delete(offset);
  state.order = state.order.filter((value) => value !== offset);
  if (preserveScroll) {
    el.reader.scrollTop = Math.max(0, el.reader.scrollTop - height);
  }
}

function trimWindow(direction) {
  while (state.order.length > state.maxMountedPages) {
    const visibleOffset = getVisiblePageOffset();
    if (direction === "up") {
      const offset = state.order[state.order.length - 1];
      if (offset === visibleOffset) break;
      removePage(offset, false);
    } else {
      const offset = state.order[0];
      if (offset === visibleOffset) break;
      removePage(offset, true);
    }
  }
}

async function loadNext() {
  const last = getLastPage();
  if (!last || last.eof || state.loadingNext) return;

  const key = `next:${last.endOffset}`;
  if (state.pending.has(key)) return;
  state.pending.add(key);
  state.loadingNext = true;
  try {
    const page = await ReadNextPage(last.endOffset);
    mountPage(page);
    trimWindow("down");
    updateVisibleProgress();
  } finally {
    state.pending.delete(key);
    state.loadingNext = false;
  }
}

async function loadPrevious() {
  const first = getFirstPage();
  if (!first || first.startOffset <= 0 || state.loadingPrev) return;

  const key = `prev:${first.startOffset}`;
  if (state.pending.has(key)) return;
  state.pending.add(key);
  state.loadingPrev = true;

  const beforeHeight = el.reader.scrollHeight;
  const beforeTop = el.reader.scrollTop;
  try {
    const page = await ReadPreviousPage(first.startOffset);
    if (page.endOffset <= page.startOffset && page.startOffset === 0) return;
    mountPage(page);
    await nextFrame();
    el.reader.scrollTop = beforeTop + (el.reader.scrollHeight - beforeHeight);
    trimWindow("up");
    updateVisibleProgress();
  } finally {
    state.pending.delete(key);
    state.loadingPrev = false;
  }
}

async function prefetchNext() {
  await loadNext();
}

async function prefetchPrevious() {
  await loadPrevious();
}

async function prefetchAround() {
  await prefetchNext();
  await prefetchPrevious();
}

function onReaderScroll() {
  if (!state.meta || state.suppressScroll) return;

  const top = el.reader.scrollTop;
  const bottomDistance =
    el.reader.scrollHeight - el.reader.clientHeight - el.reader.scrollTop;

  if (bottomDistance < LOAD_THRESHOLD_PX) {
    run(loadNext);
  } else if (bottomDistance < PREFETCH_THRESHOLD_PX) {
    run(prefetchNext);
  }

  if (top < LOAD_THRESHOLD_PX) {
    run(loadPrevious);
  } else if (top < PREFETCH_THRESHOLD_PX) {
    run(prefetchPrevious);
  }

  updateVisibleProgress();
  debouncedSaveProgress();
}

async function jump() {
  if (!state.meta) return;
  const value = el.jumpInput.value.trim();
  if (!value) return;
  setBusy(t("status.jumping"));
  let page;
  if (value.endsWith("%")) {
    page = await JumpToPercent(Number(value.slice(0, -1)));
  } else {
    page = await JumpToOffset(Number(value));
  }
  await showAnchorPage(page);
  setStatus(t("status.jumpedOffset", { offset: page.startOffset }));
}

async function search() {
  if (!state.meta || state.searching) return;
  const keyword = el.searchInput.value.trim();
  if (!keyword) return;

  const seq = ++state.searchSeq;
  state.searching = true;
  el.searchButton.disabled = true;
  try {
    if (keyword !== state.searchResultsKeyword || !state.searchSessionId) {
      await loadSearchResults(keyword, seq);
      if (seq !== state.searchSeq || !state.searchTotal) return;
      await goToSearchHitIndex(0);
      setStatus(t("status.found", { total: state.searchTotal }));
      return;
    }

    if (state.searchTotal) {
      await goRelativeSearchHit(1);
      return;
    }

    setStatus(t("search.noMatch"), true);
  } finally {
    if (seq === state.searchSeq) {
      state.searching = false;
      el.searchButton.disabled = false;
    }
  }
}

async function loadSearchResults(keyword, seq) {
  state.searchStatsLoading = true;
  state.searchSessionId = "";
  state.searchResultsKeyword = keyword;
  state.searchPreviewCache.clear();
  state.searchPreviewPending.clear();
  state.searchTotal = 0;
  state.searchCurrentIndex = -1;
  state.searchPanelVisible = true;
  state.searchMatch = null;
  el.searchResults.scrollTop = 0;
  renderSearchResults();
  setBusy(t("status.searching"));

  const summary = await StartSearch(keyword);
  if (seq !== state.searchSeq) return;
  state.searchSessionId = summary.searchId;
  state.searchResultsKeyword = summary.keyword || keyword;
  state.searchTotal = summary.total || 0;
  state.searchStatsLoading = false;
  renderSearchResults();
  if (state.searchTotal) {
    ensureSearchVisibleRange();
  } else {
    setStatus(t("search.noMatch"), true);
  }
}

async function goRelativeSearchHit(step) {
  if (!state.searchTotal) {
    await search();
    return;
  }
  const current = state.searchCurrentIndex < 0 ? -1 : state.searchCurrentIndex;
  const next =
    current < 0 ? 0 : (current + step + state.searchTotal) % state.searchTotal;
  await goToSearchHitIndex(next);
  setStatus(t("status.jumpedHit", { index: next + 1 }));
}

function hideSearchResults() {
  state.searchPanelVisible = false;
  renderSearchResults();
}

function resetSearchStateForFileChange() {
  state.searchSeq++;
  state.searching = false;
  state.searchSessionId = "";
  state.searchPreviewCache.clear();
  state.searchPreviewPending.clear();
  state.searchTotal = 0;
  state.searchCurrentIndex = -1;
  state.searchResultsKeyword = "";
  state.searchStatsLoading = false;
  state.searchPanelVisible = false;
  state.searchMatch = null;
  state.lastSearchKeyword = "";
  state.lastSearchHitOffset = -1;
  state.lastSearchHitByteLength = 0;
  el.searchButton.disabled = false;
  el.searchResults.scrollTop = 0;
}

async function goToSearchHitIndex(index) {
  if (!state.searchSessionId || index < 0 || index >= state.searchTotal) return;
  const result = await SearchHitPageByIndex(state.searchSessionId, index);
  state.searchCurrentIndex = index;
  state.searchMatch = {
    keyword: state.searchResultsKeyword,
    pageStartOffset: result.page.startOffset,
    hitOffset: result.hitOffset,
    hitByteLength: result.hitByteLength,
    lineIndex: result.lineIndex,
    lineCharStart: result.lineCharStart,
    lineCharEnd: result.lineCharEnd,
  };
  state.lastSearchKeyword = state.searchResultsKeyword;
  state.lastSearchHitOffset = result.hitOffset;
  state.lastSearchHitByteLength = result.hitByteLength;
  scrollSearchListToIndex(index);
  await showAnchorPage(result.page, state.searchMatch);
  await nextFrame();
  await nextFrame();
  scrollToSearchMatch(result.page.startOffset);
  renderSearchResults();
}

async function addBookmark() {
  if (!state.meta) return;
  await AddBookmark(el.bookmarkName.value.trim(), getVisiblePageOffset());
  state.bookmarks = await ListBookmarks();
  el.bookmarkName.value = "";
  renderBookmarks();
  setSidePanelTab("bookmarks");
  setStatus(t("status.bookmarkSaved"));
}

function renderPageText(pageEl, page) {
  const match = state.searchMatch;
  if (
    !match ||
    page.startOffset !== match.pageStartOffset ||
    match.lineIndex < 0
  ) {
    pageEl.textContent = page.lines.length ? `${page.lines.join("\n")}\n` : "";
    return;
  }

  const html = page.lines
    .map((line, index) => {
      if (index !== match.lineIndex) return escapeHtml(line);

      const start = codePointIndexToStringIndex(line, match.lineCharStart);
      const end = codePointIndexToStringIndex(line, match.lineCharEnd);
      return `${escapeHtml(line.slice(0, start))}<mark class="search-hit">${escapeHtml(line.slice(start, end))}</mark>${escapeHtml(line.slice(end))}`;
    })
    .join("\n");
  pageEl.innerHTML = page.lines.length ? `${html}\n` : "";
}

function scrollToSearchMatch(pageStartOffset = null) {
  const hit = el.pageContainer.querySelector(".search-hit");
  if (!hit) {
    if (pageStartOffset !== null) scrollPageToTop(pageStartOffset);
    return;
  }
  hit.scrollIntoView({ block: "center", inline: "nearest" });
}

function syncCurrentFileToList(meta) {
  if (!meta?.path) return;
  const exists = state.folderFiles.some((file) => file.path === meta.path);
  if (exists) return;
  state.folderFiles = [
    {
      path: meta.path,
      name: meta.name || fileNameFromPath(meta.path),
      size: meta.size || 0,
      modTime: meta.modTime || 0,
    },
  ];
}

function renderFolderFiles() {
  el.folderFiles.innerHTML = "";
  if (!state.folderFiles.length) {
    el.folderFiles.innerHTML = `<div class="empty-folder-files">${escapeHtml(t("files.empty"))}</div>`;
    return;
  }
  state.folderFiles.forEach((file) => {
    const button = document.createElement("button");
    button.className = `folder-file${file.path === state.path ? " active" : ""}`;
    button.innerHTML = `<span class="folder-file-name">${escapeHtml(file.name)}</span><span class="folder-file-meta">${formatFileSize(file.size)}</span>`;
    button.addEventListener("click", () =>
      run(async () => {
        state.path = file.path;
        el.filePath.value = file.path;
        await openCurrentFile();
        renderFolderFiles();
      }),
    );
    el.folderFiles.appendChild(button);
  });
}

function fileNameFromPath(path) {
  return String(path).split(/[\\/]/).pop() || path;
}

function formatFileSize(size) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`;
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function setSidePanelTab(tab) {
  state.sidePanelTab = tab === "bookmarks" ? "bookmarks" : "files";
  const filesActive = state.sidePanelTab === "files";
  el.filesTab.classList.toggle("active", filesActive);
  el.bookmarksTab.classList.toggle("active", !filesActive);
  el.filesPanel.classList.toggle("active", filesActive);
  el.bookmarksPanel.classList.toggle("active", !filesActive);
}

function renderSearchResults() {
  el.searchPanel.classList.toggle("open", state.searchPanelVisible);
  if (state.searchStatsLoading) {
    el.searchSummary.textContent = t("search.counting");
    el.searchResults.innerHTML = `<div class="empty-search-results">${escapeHtml(t("search.scanning"))}</div>`;
    return;
  }
  if (!state.searchResultsKeyword) {
    el.searchSummary.textContent = t("search.none");
    el.searchResults.innerHTML = `<div class="empty-search-results">${escapeHtml(t("search.prompt"))}</div>`;
    return;
  }
  if (!state.searchTotal) {
    el.searchSummary.textContent = t("search.noMatch");
    el.searchResults.innerHTML = `<div class="empty-search-results">${escapeHtml(t("search.noResult"))}</div>`;
    return;
  }

  el.searchSummary.textContent =
    state.searchCurrentIndex >= 0
      ? t("search.current", {
          current: state.searchCurrentIndex + 1,
          total: state.searchTotal,
        })
      : t("search.total", { total: state.searchTotal });
  renderSearchVirtualRows();
  ensureSearchVisibleRange();
}

function renderSearchVirtualRows() {
  const { first, last } = getSearchVisibleRange();
  const rowHeight = state.searchVirtual.rowHeight;
  const totalHeight = state.searchTotal * rowHeight;
  const rows = [];
  for (let index = first; index <= last; index++) {
    const hit = state.searchPreviewCache.get(index);
    if (hit) {
      rows.push(`
        <button class="search-result${index === state.searchCurrentIndex ? " active" : ""}" data-index="${index}" style="height:${rowHeight}px">
          <span class="search-result-meta">#${index + 1} · ${t("search.line")} ${hit.lineNumber} · offset ${hit.offset}</span>
          <span class="search-result-preview">${renderSearchPreview(hit)}</span>
        </button>
      `);
    } else {
      rows.push(`
        <button class="search-result loading${index === state.searchCurrentIndex ? " active" : ""}" data-index="${index}" style="height:${rowHeight}px">
          <span class="search-result-meta">#${index + 1}</span>
          <span class="search-result-preview">${escapeHtml(t("search.loading"))}</span>
        </button>
      `);
    }
  }
  el.searchResults.innerHTML = `
    <div class="search-virtual-spacer" style="height:${totalHeight}px">
      <div class="search-virtual-window" style="transform:translateY(${first * rowHeight}px)">
        ${rows.join("")}
      </div>
    </div>
  `;
  el.searchResults.querySelectorAll(".search-result").forEach((button) => {
    button.addEventListener("click", () =>
      run(async () => {
        const index = Number(button.dataset.index);
        await goToSearchHitIndex(index);
        setStatus(t("status.jumpedHit", { index: index + 1 }));
      }),
    );
  });
}

function getSearchVisibleRange() {
  if (!state.searchTotal) return { first: 0, last: -1 };
  const rowHeight = state.searchVirtual.rowHeight;
  const overscan = state.searchVirtual.overscan;
  const viewport = el.searchResults.clientHeight || rowHeight * 6;
  const first = Math.max(
    0,
    Math.floor(el.searchResults.scrollTop / rowHeight) - overscan,
  );
  const count = Math.ceil(viewport / rowHeight) + overscan * 2;
  const last = Math.min(state.searchTotal - 1, first + count - 1);
  return { first, last };
}

function ensureSearchVisibleRange() {
  const { first, last } = getSearchVisibleRange();
  if (last < first) return;
  ensureSearchPreviewRange(first, last);
}

function ensureSearchPreviewRange(first, last) {
  if (!state.searchSessionId) return;
  const pageSize = state.searchVirtual.pageSize;
  const startPage = Math.max(0, Math.floor(first / pageSize) - 1);
  const endPage = Math.floor(last / pageSize) + 1;
  for (let page = startPage; page <= endPage; page++) {
    const offset = page * pageSize;
    if (offset >= state.searchTotal) continue;
    const key = `${state.searchSessionId}:${offset}`;
    if (
      state.searchPreviewPending.has(key) ||
      isSearchPreviewPageCached(offset, pageSize)
    )
      continue;
    state.searchPreviewPending.add(key);
    const seq = state.searchSeq;
    SearchHitPreviews(state.searchSessionId, offset, pageSize)
      .then((result) => {
        if (
          seq !== state.searchSeq ||
          result.searchId !== state.searchSessionId
        )
          return;
        (result.hits || []).forEach((hit) =>
          state.searchPreviewCache.set(hit.index, hit),
        );
        trimSearchPreviewCache();
        renderSearchResults();
      })
      .catch((error) => setStatus(error?.message || String(error), true))
      .finally(() => state.searchPreviewPending.delete(key));
  }
}

function isSearchPreviewPageCached(offset, limit) {
  const end = Math.min(state.searchTotal, offset + limit);
  for (let index = offset; index < end; index++) {
    if (!state.searchPreviewCache.has(index)) return false;
  }
  return true;
}

function scrollSearchListToIndex(index) {
  if (!state.searchPanelVisible) return;
  const rowHeight = state.searchVirtual.rowHeight;
  const top = index * rowHeight;
  const bottom = top + rowHeight;
  const visibleTop = el.searchResults.scrollTop;
  const visibleBottom = visibleTop + el.searchResults.clientHeight;
  if (top < visibleTop) {
    el.searchResults.scrollTop = top;
  } else if (bottom > visibleBottom) {
    el.searchResults.scrollTop = Math.max(
      0,
      bottom - el.searchResults.clientHeight,
    );
  }
}

function trimSearchPreviewCache() {
  if (state.searchPreviewCache.size <= state.searchVirtual.maxCachedRows)
    return;
  const { first, last } = getSearchVisibleRange();
  const keepStart = Math.max(0, first - state.searchVirtual.pageSize * 4);
  const keepEnd = Math.min(
    state.searchTotal - 1,
    last + state.searchVirtual.pageSize * 4,
  );
  for (const index of state.searchPreviewCache.keys()) {
    if (
      index !== state.searchCurrentIndex &&
      (index < keepStart || index > keepEnd)
    ) {
      state.searchPreviewCache.delete(index);
    }
  }
}

function renderSearchPreview(hit) {
  const line = hit.linePreview || "";
  const start = codePointIndexToStringIndex(line, hit.lineCharStart);
  const end = codePointIndexToStringIndex(line, hit.lineCharEnd);
  return `${escapeHtml(line.slice(0, start))}<mark class="search-hit mini">${escapeHtml(line.slice(start, end))}</mark>${escapeHtml(line.slice(end))}`;
}

function renderBookmarks() {
  el.bookmarksTab.textContent = state.bookmarks.length
    ? t("bookmark.count", { count: state.bookmarks.length })
    : t("tabs.bookmarks");
  el.bookmarks.innerHTML = "";
  if (!state.bookmarks.length) {
    el.bookmarks.innerHTML = `<div class="empty-bookmarks">${escapeHtml(t("bookmark.empty"))}</div>`;
    return;
  }
  state.bookmarks.forEach((bookmark, index) => {
    const item = document.createElement("div");
    item.className = "bookmark";
    item.innerHTML = `
      <button class="bookmark-main" type="button">
        <span class="bookmark-name">${index + 1}. ${escapeHtml(bookmark.name)}</span>
        <span class="bookmark-offset">${bookmark.offset}</span>
      </button>
      <button class="bookmark-delete" type="button" title="${escapeHtml(t("bookmark.deleteTitle"))}">${escapeHtml(t("bookmark.delete"))}</button>
    `;
    item.querySelector(".bookmark-main").addEventListener("click", () =>
      run(async () => {
        const page = await GoToBookmark(index);
        await showAnchorPage(page);
        setStatus(t("status.bookmarkJumped"));
      }),
    );
    item.querySelector(".bookmark-delete").addEventListener("click", () =>
      run(async () => {
        await DeleteBookmark(index);
        state.bookmarks = await ListBookmarks();
        renderBookmarks();
        setStatus(t("status.bookmarkDeleted"));
      }),
    );
    el.bookmarks.appendChild(item);
  });
}

function getFirstPage() {
  return state.order.length ? state.pages.get(state.order[0]) : null;
}

function getLastPage() {
  return state.order.length
    ? state.pages.get(state.order[state.order.length - 1])
    : null;
}

function getPageElement(offset) {
  return el.pageContainer.querySelector(`[data-start="${offset}"]`);
}

function getVisiblePageOffset() {
  if (!state.order.length) return 0;
  const readerRect = el.reader.getBoundingClientRect();
  const pageEls = [...el.pageContainer.querySelectorAll(".page")];
  for (const pageEl of pageEls) {
    const rect = pageEl.getBoundingClientRect();
    if (
      rect.bottom >= readerRect.top + 24 &&
      rect.top <= readerRect.bottom - 24
    ) {
      return Number(pageEl.dataset.start);
    }
  }
  return state.order[0];
}

function scrollPageToTop(offset) {
  const pageEl = getPageElement(offset);
  if (!pageEl) return;
  state.suppressScroll = true;
  el.reader.scrollTop = pageEl.offsetTop - el.pageContainer.offsetTop;
  requestAnimationFrame(() => {
    state.suppressScroll = false;
  });
}

function updateVisibleProgress() {
  if (!state.meta) {
    el.progressText.textContent = "0.0000%";
    el.progressBar.style.width = "0%";
    el.offsetChip.textContent = "offset 0";
    el.encodingChip.textContent = "encoding -";
    el.stateChip.textContent = t("state.idle");
    return;
  }
  const offset = getVisiblePageOffset();
  const page = state.pages.get(offset) || getFirstPage();
  if (!page) {
    el.progressText.textContent = "0.0000%";
    el.progressBar.style.width = "0%";
    return;
  }
  const size = page.fileSize || state.meta.size || 0;
  const atVisualBottom =
    el.reader.scrollHeight <= el.reader.clientHeight ||
    el.reader.scrollTop + el.reader.clientHeight >= el.reader.scrollHeight - 2;
  const progressOffset =
    page.eof && atVisualBottom
      ? size
      : Math.max(
          page.startOffset,
          Math.min(page.endOffset || page.startOffset, size),
        );
  const percent = size === 0 ? 100 : (progressOffset / size) * 100;
  const clampedPercent = Math.min(100, Math.max(0, percent));
  el.progressText.textContent = `${clampedPercent.toFixed(4)}%`;
  el.progressBar.style.width = `${clampedPercent}%`;
  el.offsetChip.textContent = `${progressOffset} / ${size}`;
  el.encodingChip.textContent = page.encoding;
  el.stateChip.textContent = page.eof
    ? "EOF"
    : page.bof
      ? "BOF"
      : page.truncated
        ? t("state.truncated")
        : t("state.reading");
}

function debouncedSaveProgress() {
  clearTimeout(state.saveTimer);
  state.saveTimer = setTimeout(() => {
    const offset = getVisiblePageOffset();
    if (state.meta && offset >= 0) {
      SaveProgress(offset).catch(console.error);
    }
  }, 750);
}

async function run(fn) {
  try {
    await fn();
  } catch (error) {
    console.error(error);
    setStatus(error?.message || String(error), true);
  }
}

function setBusy(message) {
  el.statusText.textContent = message;
  el.statusText.classList.remove("error");
}

function setStatus(message, isError = false) {
  el.statusText.textContent = message;
  el.statusText.classList.toggle("error", isError);
  updateVisibleProgress();
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function codePointIndexToStringIndex(value, codePointIndex) {
  if (codePointIndex <= 0) return 0;
  let index = 0;
  let count = 0;
  for (const char of value) {
    if (count >= codePointIndex) break;
    index += char.length;
    count += 1;
  }
  return index;
}

function nextFrame() {
  return new Promise((resolve) => requestAnimationFrame(resolve));
}
