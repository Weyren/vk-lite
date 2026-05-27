const state = {
  mode: "login",
  token: localStorage.getItem("vk-lite-token") || "",
  userId: Number(localStorage.getItem("vk-lite-user-id") || 0),
  email: localStorage.getItem("vk-lite-email") || "",
  currentProfileId: 0,
  currentView: "feed",
  feedPage: 1,
  profilePostsPage: 1,
  feedPerPage: 10,
  autoRefreshTimer: null,
};

const els = {
  homeBtn: document.querySelector("#homeBtn"),
  navFeedBtn: document.querySelector("#navFeedBtn"),
  navMeBtn: document.querySelector("#navMeBtn"),
  authModal: document.querySelector("#authModal"),
  authForm: document.querySelector("#authForm"),
  loginTab: document.querySelector("#loginTab"),
  registerTab: document.querySelector("#registerTab"),
  loginOpenBtn: document.querySelector("#loginOpenBtn"),
  nameField: document.querySelector("#nameField"),
  nameInput: document.querySelector("#nameInput"),
  emailInput: document.querySelector("#emailInput"),
  passwordInput: document.querySelector("#passwordInput"),
  authSubmitText: document.querySelector("#authSubmitText"),
  sessionStatus: document.querySelector("#sessionStatus"),
  logoutBtn: document.querySelector("#logoutBtn"),
  composerSection: document.querySelector("#composerSection"),
  postForm: document.querySelector("#postForm"),
  postContent: document.querySelector("#postContent"),
  mediaUrl: document.querySelector("#mediaUrl"),
  mediaFile: document.querySelector("#mediaFile"),
  mediaFileLabel: document.querySelector("#mediaFileLabel"),
  profileForm: document.querySelector("#profileForm"),
  profileId: document.querySelector("#profileId"),
  refreshFeedBtn: document.querySelector("#refreshFeedBtn"),
  profileView: document.querySelector("#profileView"),
  feedList: document.querySelector("#feedList"),
  feedMeta: document.querySelector("#feedMeta"),
  feedTitle: document.querySelector("#feedTitle"),
  autoRefreshState: document.querySelector("#autoRefreshState"),
  loadMoreFeedBtn: document.querySelector("#loadMoreFeedBtn"),
  toast: document.querySelector("#toast"),
};

function init() {
  els.homeBtn.addEventListener("click", showFeed);
  els.navFeedBtn.addEventListener("click", showFeed);
  els.navMeBtn.addEventListener("click", () => {
    if (ensureAuth()) navigateToProfile(state.userId);
  });
  els.loginOpenBtn.addEventListener("click", showAuthModal);
  els.loginTab.addEventListener("click", () => setMode("login"));
  els.registerTab.addEventListener("click", () => setMode("register"));
  els.authForm.addEventListener("submit", handleAuth);
  els.logoutBtn.addEventListener("click", logout);
  els.postForm.addEventListener("submit", createPost);
  els.profileForm.addEventListener("submit", openProfileFromSearch);
  els.refreshFeedBtn.addEventListener("click", () => loadFeed({ reset: true, manual: true }));
  els.loadMoreFeedBtn.addEventListener("click", loadMoreFeed);
  els.mediaFile.addEventListener("change", renderSelectedMedia);
  window.addEventListener("hashchange", handleRoute);

  renderSession();
  renderEmptyFeed();
  handleRoute();

  if (state.token) {
    loadFeed({ reset: true });
  } else {
    showAuthModal();
  }
  startAutoRefresh();
  refreshIcons();
}

function setMode(mode) {
  state.mode = mode;
  const isRegister = mode === "register";
  els.loginTab.classList.toggle("active", !isRegister);
  els.registerTab.classList.toggle("active", isRegister);
  els.nameField.classList.toggle("hidden", !isRegister);
  els.nameInput.required = isRegister;
  els.authSubmitText.textContent = isRegister ? "Создать аккаунт" : "Войти";
  refreshIcons();
}

async function handleAuth(event) {
  event.preventDefault();

  const email = els.emailInput.value.trim();
  const password = els.passwordInput.value;
  const endpoint = state.mode === "register" ? "/api/v1/users" : "/api/v1/auth/login";
  const payload = { email, password };

  if (state.mode === "register") {
    payload.name = els.nameInput.value.trim();
  }

  const result = await request(endpoint, { method: "POST", body: payload, auth: false });
  if (!result.ok) {
    return showToast(result.error, true);
  }

  if (state.mode === "register") {
    showToast(`Пользователь #${result.data.id} создан. Теперь можно войти.`);
    setMode("login");
    return;
  }

  state.token = result.data.access_token;
  state.userId = readUserIDFromToken(state.token);
  state.email = email;
  localStorage.setItem("vk-lite-token", state.token);
  localStorage.setItem("vk-lite-user-id", String(state.userId));
  localStorage.setItem("vk-lite-email", state.email);

  hideAuthModal();
  renderSession();
  showToast("Вход выполнен");
  loadFeed({ reset: true });
  startAutoRefresh();
}

async function createPost(event) {
  event.preventDefault();
  if (!ensureAuth()) return;

  const content = els.postContent.value.trim();
  let mediaUrl = els.mediaUrl.value.trim();

  if (els.mediaFile.files.length > 0) {
    const upload = await uploadMedia(els.mediaFile.files[0]);
    if (!upload.ok) {
      return showToast(upload.error, true);
    }
    mediaUrl = upload.data.url;
  }

  const result = await request("/api/v1/posts", { method: "POST", body: { content, media_url: mediaUrl } });
  if (!result.ok) {
    return showToast(result.error, true);
  }

  els.postContent.value = "";
  els.mediaUrl.value = "";
  els.mediaFile.value = "";
  renderSelectedMedia();
  showToast(`Пост #${result.data.post.id} опубликован`);
  loadFeed({ reset: true });
}

async function uploadMedia(file) {
  const formData = new FormData();
  formData.append("file", file);

  try {
    const response = await fetch("/api/v1/media", {
      method: "POST",
      headers: { Authorization: `Bearer ${state.token}` },
      body: formData,
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      return { ok: false, error: data.error || `HTTP ${response.status}` };
    }
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error: error.message };
  }
}

function renderSelectedMedia() {
  const file = els.mediaFile.files[0];
  els.mediaFileLabel.textContent = file ? file.name : "Прикрепить фото или видео";
}

function openProfileFromSearch(event) {
  event.preventDefault();
  if (!ensureAuth()) return;

  const id = Number(els.profileId.value);
  if (!id) {
    return showToast("Укажи ID профиля", true);
  }

  navigateToProfile(id);
}

function navigateToProfile(id) {
  window.location.hash = `profile/${id}`;
}

async function handleRoute() {
  const match = window.location.hash.match(/^#profile\/(\d+)$/);
  if (!match) {
    showFeed(false);
    return;
  }

  if (!ensureAuth()) return;
  await loadProfile(Number(match[1]));
}

async function loadProfile(id) {
  const result = await request(`/api/v1/users/${id}`);
  if (!result.ok) {
    return showToast(result.error, true);
  }

  state.currentView = "profile";
  state.currentProfileId = id;
  state.profilePostsPage = 1;
  setActiveNav(id === state.userId ? "me" : "");
  renderProfile(result.data.user);
  await loadProfilePosts(id, { reset: true });
  window.scrollTo({ top: 0, behavior: "smooth" });
}

async function loadProfilePosts(id, options = {}) {
  const page = options.reset ? 1 : state.profilePostsPage;
  const result = await request(`/api/v1/users/${id}/posts?page=${page}&per_page=${state.feedPerPage}`);
  if (!result.ok) {
    return showToast(result.error, true);
  }

  state.profilePostsPage = page;
  const container = els.profileView.querySelector("[data-profile-posts]");
  if (container) {
    container.innerHTML = renderPostsHTML(result.data.posts || []);
    bindPostActions(container);
  }
  const moreButton = els.profileView.querySelector("[data-profile-more]");
  if (moreButton) {
    moreButton.classList.toggle("hidden", (result.data.posts || []).length < state.feedPerPage);
    moreButton.onclick = async () => {
      state.profilePostsPage += 1;
      const more = await request(`/api/v1/users/${id}/posts?page=${state.profilePostsPage}&per_page=${state.feedPerPage}`);
      if (!more.ok) return showToast(more.error, true);
      container.insertAdjacentHTML("beforeend", renderPostsHTML(more.data.posts || []));
      bindPostActions(container);
      moreButton.classList.toggle("hidden", (more.data.posts || []).length < state.feedPerPage);
    };
  }
}

async function toggleFollowFromProfile(id, isFollowing) {
  if (!ensureAuth()) return;

  const result = await request(`/api/v1/users/${id}/follow`, { method: "POST" });
  if (!result.ok) {
    return showToast(result.error, true);
  }

  showToast(isFollowing ? "Подписка удалена" : "Подписка оформлена");
  await loadProfile(id);
  loadFeed({ reset: true });
}

async function loadFeed(options = {}) {
  if (!state.token) {
    renderEmptyFeed();
    return;
  }

  if (options.reset) {
    state.feedPage = 1;
  }

  const result = await request(`/api/v1/feed?page=${state.feedPage}&per_page=${state.feedPerPage}`);
  if (!result.ok) {
    return showToast(result.error, true);
  }

  els.feedTitle.textContent = "Лента";
  els.feedMeta.textContent = result.data.from_cache ? "Загружено из Redis-кеша. Автообновление каждые 10 секунд." : "Загружено из PostgreSQL. Автообновление каждые 10 секунд.";
  els.feedList.innerHTML = renderPostsHTML(result.data.posts || []);
  bindPostActions(els.feedList);
  els.loadMoreFeedBtn.classList.toggle("hidden", (result.data.posts || []).length < state.feedPerPage);
  if (options.manual) showToast("Лента обновлена");
}

async function loadMoreFeed() {
  state.feedPage += 1;
  const result = await request(`/api/v1/feed?page=${state.feedPage}&per_page=${state.feedPerPage}`);
  if (!result.ok) {
    state.feedPage -= 1;
    return showToast(result.error, true);
  }
  els.feedList.insertAdjacentHTML("beforeend", renderPostsHTML(result.data.posts || []));
  bindPostActions(els.feedList);
  els.loadMoreFeedBtn.classList.toggle("hidden", (result.data.posts || []).length < state.feedPerPage);
}

async function likePost(postId, button) {
  const result = await request(`/api/v1/posts/${postId}/like`, { method: "POST" });
  if (!result.ok) {
    return showToast(result.error, true);
  }

  button.classList.toggle("active", result.data.liked);
  button.querySelector("span").textContent = `${result.data.likes_count}`;
  refreshIcons();
}

async function request(url, options = {}) {
  const headers = { "Content-Type": "application/json" };
  if (options.auth !== false && state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }

  try {
    const response = await fetch(url, {
      method: options.method || "GET",
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      return { ok: false, error: data.error || `HTTP ${response.status}` };
    }
    return { ok: true, data };
  } catch (error) {
    return { ok: false, error: error.message };
  }
}

function renderSession() {
  const loggedIn = Boolean(state.token);
  els.sessionStatus.textContent = loggedIn ? `${state.email || "Пользователь"} · ID ${state.userId || "?"}` : "Гость";
  els.logoutBtn.classList.toggle("hidden", !loggedIn);
  els.loginOpenBtn.classList.toggle("hidden", loggedIn);
  els.composerSection.classList.toggle("hidden", !loggedIn);
  refreshIcons();
}

function renderProfile(user) {
  const isMe = Number(user.id) === Number(state.userId);
  const initials = (user.name || user.email || "?").trim().slice(0, 2).toUpperCase();
  const followButton = isMe ? "" : `
    <button class="primary-button ${user.is_following ? "danger" : ""}" type="button" data-follow-profile="${user.id}" data-following="${user.is_following}">
      <i data-lucide="${user.is_following ? "user-minus" : "user-plus"}"></i>
      ${user.is_following ? "Отписаться" : "Подписаться"}
    </button>
  `;

  els.profileView.classList.remove("hidden");
  els.profileView.innerHTML = `
    <div class="profile-main">
      <div class="profile-identity">
        <div class="avatar">${escapeHTML(initials)}</div>
        <div>
          <div class="profile-name">${escapeHTML(user.name || "Без имени")}</div>
          <div class="profile-meta">${escapeHTML(user.email)} · ID ${user.id}</div>
        </div>
      </div>
      ${followButton}
    </div>
    <div class="profile-grid">
      <div class="metric"><strong>${user.followers_count}</strong><span>подписчиков</span></div>
      <div class="metric"><strong>${user.following_count}</strong><span>подписок</span></div>
      <div class="metric"><strong>${user.posts_count}</strong><span>постов</span></div>
    </div>
    <h2 class="profile-posts-title">Посты пользователя</h2>
    <div class="profile-posts" data-profile-posts>
      <div class="empty">Загружаем посты...</div>
    </div>
    <button class="secondary-button hidden" type="button" data-profile-more>
      <i data-lucide="chevrons-down"></i>
      Загрузить еще
    </button>
  `;

  const button = els.profileView.querySelector("[data-follow-profile]");
  if (button) {
    button.addEventListener("click", () => toggleFollowFromProfile(user.id, button.dataset.following === "true"));
  }
  refreshIcons();
}

function renderPostsHTML(posts) {
  if (!posts.length) {
    return `<div class="empty">Постов пока нет.</div>`;
  }

  return posts.map((post) => `
    <article class="post">
      <div class="post-head">
        <div>
          <button class="author-button" type="button" data-profile-id="${post.author_id}">
            ${escapeHTML(post.author_name || `User #${post.author_id}`)}
          </button>
          <div class="post-time">Post #${post.id} · ${formatDate(post.created_at)}</div>
        </div>
      </div>
      <div class="post-content">${escapeHTML(post.content || "")}</div>
      ${renderMedia(post.media_url)}
      <div class="post-actions">
        <button class="icon-button like-button" type="button" data-post-id="${post.id}" title="Лайк">
          <i data-lucide="heart"></i>
          <span>${post.likes_count}</span>
        </button>
      </div>
    </article>
  `).join("");
}

function renderMedia(url) {
  if (!url) return "";
  const safeURL = escapeAttr(url);
  const lower = url.toLowerCase();
  if (/\.(png|jpg|jpeg|gif|webp)(\?|$)/.test(lower)) {
    return `<img class="post-media-preview" src="${safeURL}" alt="Медиа поста" loading="lazy" />`;
  }
  if (/\.(mp4|webm|ogg)(\?|$)/.test(lower)) {
    return `<video class="post-media-preview" src="${safeURL}" controls></video>`;
  }
  return `<a class="post-media-link" href="${safeURL}" target="_blank" rel="noreferrer">${escapeHTML(url)}</a>`;
}

function bindPostActions(root) {
  root.querySelectorAll("[data-post-id]").forEach((button) => {
    button.onclick = () => likePost(button.dataset.postId, button);
  });
  root.querySelectorAll("[data-profile-id]").forEach((button) => {
    button.onclick = () => navigateToProfile(button.dataset.profileId);
  });
  refreshIcons();
}

function renderEmptyFeed() {
  els.feedMeta.textContent = "Войди, чтобы увидеть посты.";
  els.feedList.innerHTML = `<div class="empty">После входа здесь появится лента новостей.</div>`;
  els.loadMoreFeedBtn.classList.add("hidden");
}

function showFeed(updateHash = true) {
  if (updateHash) {
    window.location.hash = "";
  }
  state.currentView = "feed";
  state.currentProfileId = 0;
  els.profileView.classList.add("hidden");
  setActiveNav("feed");
  if (state.token) loadFeed({ reset: true });
}

function setActiveNav(item) {
  els.navFeedBtn.classList.toggle("active", item === "feed");
  els.navMeBtn.classList.toggle("active", item === "me");
}

function logout() {
  state.token = "";
  state.userId = 0;
  state.email = "";
  state.currentProfileId = 0;
  state.currentView = "feed";
  localStorage.removeItem("vk-lite-token");
  localStorage.removeItem("vk-lite-user-id");
  localStorage.removeItem("vk-lite-email");
  els.profileView.classList.add("hidden");
  stopAutoRefresh();
  renderSession();
  renderEmptyFeed();
  showAuthModal();
  showToast("Ты вышел из аккаунта");
}

function ensureAuth() {
  if (state.token) return true;
  showAuthModal();
  showToast("Сначала войди в аккаунт", true);
  return false;
}

function showAuthModal() {
  els.authModal.classList.remove("hidden");
  refreshIcons();
}

function hideAuthModal() {
  els.authModal.classList.add("hidden");
}

function startAutoRefresh() {
  stopAutoRefresh();
  if (!state.token) return;

  state.autoRefreshTimer = window.setInterval(() => {
    if (!state.token) return;
    if (state.currentView === "profile" && state.currentProfileId) {
      loadProfilePosts(state.currentProfileId, { reset: true });
      return;
    }
    loadFeed({ reset: true });
  }, 10000);
  els.autoRefreshState.textContent = "автообновление 10 сек";
}

function stopAutoRefresh() {
  if (state.autoRefreshTimer) {
    window.clearInterval(state.autoRefreshTimer);
    state.autoRefreshTimer = null;
  }
  els.autoRefreshState.textContent = "автообновление";
}

function readUserIDFromToken(token) {
  try {
    const payload = JSON.parse(atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")));
    return Number(payload.sub || 0);
  } catch {
    return 0;
  }
}

function showToast(message, isError = false) {
  els.toast.textContent = message;
  els.toast.classList.toggle("error", isError);
  els.toast.classList.remove("hidden");
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => els.toast.classList.add("hidden"), 3400);
}

function formatDate(value) {
  if (!value) return "";
  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(new Date(value));
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapeAttr(value) {
  return escapeHTML(value).replaceAll("`", "&#096;");
}

function refreshIcons() {
  if (window.lucide) {
    window.lucide.createIcons();
  }
}

init();
