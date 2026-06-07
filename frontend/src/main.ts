type Summary = {
  counts: Record<string, number>;
};

type Project = {
  id: string;
  name: string;
  organization: string;
  paid_credits: number;
  promotional_credits: number;
};

type Conversation = {
  id: string;
  title: string;
  status: string;
  message_count: number;
};

type WorkspaceAsset = {
  id: string;
  conversation_id: string;
  asset_type: string;
  storage_path: string;
  mime_type: string;
  size_bytes: number;
};

type Model = {
  id?: string;
  slug: string;
  name: string;
  provider: string;
  modality: string;
  profile_slug: string;
  profile_name: string;
  status?: string;
};

type APIKey = {
  id: string;
  name: string;
  prefix: string;
  status: string;
};

type ChatResponse = {
  id?: string;
  model?: string;
  choices?: Array<{ message?: { content?: string } }>;
};

type AuthResponse = {
  user?: {
    id: string;
    email: string;
    name: string;
  };
  project_id?: string;
  provider?: string;
  session?: {
    access_token: string;
    token_type: string;
  };
};

const apiBase = (window as Window & { API_BASE_URL?: string }).API_BASE_URL || "http://localhost:8080";
let currentProjectID = "";
let currentAPIKey = "";
let models: Model[] = [];
let projects: Project[] = [];
let conversations: Conversation[] = [];
let assets: WorkspaceAsset[] = [];
let signupMode = false;
let selectedModality = "all";
let workbenchModality = "image";
let selectedWorkbenchModel = "";
const tabs = new Set(["home", "models", "model-detail", "api", "workbench", "admin", "pricing"]);
const themeModes = new Set(["system", "light", "dark"]);
const systemTheme = window.matchMedia("(prefers-color-scheme: dark)");

const mockModels: Model[] = [
  {
    slug: "mock-video",
    name: "Mock Video",
    provider: "Mock Provider",
    modality: "video",
    profile_slug: "mock-video-default",
    profile_name: "Mock Video Default"
  },
  {
    slug: "mock-audio",
    name: "Mock Audio",
    provider: "Mock Provider",
    modality: "audio",
    profile_slug: "mock-audio-default",
    profile_name: "Mock Audio Default"
  }
];

const $ = <T extends HTMLElement>(id: string): T => {
  const el = document.getElementById(id);
  if (!el) throw new Error(`missing element ${id}`);
  return el as T;
};

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#039;");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {})
    }
  });
  if (!response.ok) {
    throw new Error(`${response.status} ${await response.text()}`);
  }
  return response.json() as Promise<T>;
}

async function loadAll() {
  await Promise.all([loadSummary(), loadProjects(), loadModels()]);
}

async function loadSummary() {
  try {
    const summary = await request<Summary>("/api/v1/dev/summary");
    const counts = summary.counts || {};
    $("summary").innerHTML = [
      statCard("Models", counts.models ?? 0),
      statCard("Projects", counts.projects ?? 0),
      statCard("Usage events", counts.usage_events ?? 0)
    ].join("");
  } catch {
    $("summary").innerHTML = [
      statCard("Models", 300),
      statCard("Providers", 20),
      statCard("Modes", 5)
    ].join("");
  }
}

function statCard(label: string, value: number) {
  return `<div class="stat-card"><strong>${value}</strong><span>${escapeHTML(label)}</span></div>`;
}

async function loadProjects() {
  try {
    const data = await request<{ projects: Project[] }>("/api/v1/projects");
    projects = data.projects;
  } catch {
    projects = [];
  }
  const [project] = projects;
  if (project) currentProjectID = project.id;
  renderProjects();
  if (currentProjectID) await Promise.all([loadAPIKeys(), loadWorkspace()]);
}

function renderProjects() {
  $("projects").innerHTML =
    projects
      .map((project) => {
        const credits = project.paid_credits + project.promotional_credits;
        const active = project.id === currentProjectID ? " active" : "";
        return `<button class="workspace-row${active}" type="button" data-project-id="${escapeHTML(project.id)}">
          <span><strong>${escapeHTML(project.name)}</strong><small>${escapeHTML(project.organization)}</small></span>
          <p>${credits.toLocaleString()} credits available</p>
        </button>`;
      })
      .join("") || "<div class=\"empty-state\">No project loaded. Run the dev test data script first.</div>";
}

async function createProject() {
  const name = window.prompt("Project name");
  if (!name?.trim()) return;
  const data = await request<{ project: Project }>("/api/v1/projects", {
    method: "POST",
    body: JSON.stringify({ name: name.trim() })
  });
  projects = [...projects, data.project];
  currentProjectID = data.project.id;
  renderProjects();
  await loadWorkspace();
}

async function loadWorkspace() {
  if (!currentProjectID) {
    conversations = [];
    assets = [];
    renderWorkspaceLists();
    return;
  }
  try {
    const [conversationData, assetData] = await Promise.all([
      request<{ conversations: Conversation[] }>(`/api/v1/conversations?project_id=${encodeURIComponent(currentProjectID)}`),
      request<{ assets: WorkspaceAsset[] }>(`/api/v1/assets?project_id=${encodeURIComponent(currentProjectID)}`)
    ]);
    conversations = conversationData.conversations;
    assets = assetData.assets;
  } catch {
    conversations = [];
    assets = [];
  }
  renderWorkspaceLists();
}

function renderWorkspaceLists() {
  $("conversations").innerHTML =
    conversations
      .map((conversation) => `<button class="workspace-row" type="button" data-conversation-id="${escapeHTML(conversation.id)}">
        <span><strong>${escapeHTML(conversation.title)}</strong><small>${conversation.message_count} messages / ${escapeHTML(conversation.status)}</small></span>
      </button>`)
      .join("") || "<div class=\"empty-state\">No conversations yet.</div>";
  $("artifacts").innerHTML =
    assets
      .map((asset) => `<div class="workspace-row">
        <span><strong>${escapeHTML(asset.asset_type)}</strong><small>${escapeHTML(asset.mime_type || "artifact")} / ${asset.size_bytes.toLocaleString()} bytes</small></span>
        <p>${escapeHTML(asset.storage_path)}</p>
      </div>`)
      .join("") || "<div class=\"empty-state\">No generated artifacts yet.</div>";
}

async function createConversation() {
  if (!currentProjectID) throw new Error("Create or select a project first.");
  const title = window.prompt("Conversation title") || "New conversation";
  const data = await request<{ conversation: Conversation }>("/api/v1/conversations", {
    method: "POST",
    body: JSON.stringify({ project_id: currentProjectID, title })
  });
  conversations = [data.conversation, ...conversations];
  renderWorkspaceLists();
}

async function loadModels() {
  try {
    const data = await request<{ models: Model[] }>("/api/v1/models");
    models = data.models;
  } catch {
    models = [...mockModels];
  }
  renderModelControls();
  renderModels();
  renderMetadata();
  renderPricing();
}

function renderModelControls() {
  const usableModels = models.filter((model) => ["image", "audio", "video"].includes(model.modality));
  if (!usableModels.some((model) => model.modality === workbenchModality)) {
    workbenchModality = usableModels[0]?.modality || "image";
  }
  if (!selectedWorkbenchModel || !models.some((model) => model.profile_slug === selectedWorkbenchModel || model.slug === selectedWorkbenchModel)) {
    const defaultModel = usableModels.find((model) => model.modality === workbenchModality) || usableModels[0] || models[0];
    selectedWorkbenchModel = defaultModel ? defaultModel.profile_slug || defaultModel.slug : "mock-chat";
  }
  $("modelSelect").innerHTML =
    models
      .map((model) => {
        const value = model.profile_slug || model.slug;
        return `<option value="${escapeHTML(value)}"${value === selectedWorkbenchModel ? " selected" : ""}>${escapeHTML(model.name)}</option>`;
      })
      .join("") || "<option value=\"mock-chat\">Mock Chat</option>";
  renderWorkbenchModelPicker();
}

function renderWorkbenchModelPicker() {
  document.querySelectorAll<HTMLButtonElement>("[data-workbench-modality]").forEach((button) => {
    button.classList.toggle("active", button.dataset.workbenchModality === workbenchModality);
  });
  const query = ($("workbenchModelFilter") as HTMLInputElement).value.trim().toLowerCase();
  const visible = models.filter((model) => {
    const matchesModality = model.modality === workbenchModality;
    const matchesQuery = [model.name, model.slug, model.provider, model.profile_name].join(" ").toLowerCase().includes(query);
    return matchesModality && matchesQuery;
  });
  $("workbenchModelList").innerHTML =
    visible
      .map((model) => {
        const value = model.profile_slug || model.slug;
        return `<button class="workbench-model-option${value === selectedWorkbenchModel ? " active" : ""}" type="button" data-workbench-model="${escapeHTML(value)}">
          <span><strong>${escapeHTML(model.name)}</strong><small>${escapeHTML(model.provider)} / ${escapeHTML(model.slug)}</small></span>
          <span class="tag">${escapeHTML(priceFor(model))}</span>
        </button>`;
      })
      .join("") || "<div class=\"empty-state\">No ${escapeHTML(workbenchModality)} models match the filter.</div>";
  renderSelectedWorkbenchModel();
}

function renderSelectedWorkbenchModel() {
  const selected = models.find((model) => (model.profile_slug || model.slug) === selectedWorkbenchModel);
  $("selectedModelName").textContent = selected?.name || "No model selected";
  $("selectedModelModality").textContent = selected?.modality || "-";
  ($("modelSelect") as HTMLSelectElement).value = selectedWorkbenchModel;
}

function renderModels() {
  const query = ($("modelSearch") as HTMLInputElement).value.trim().toLowerCase();
  const visible = models.filter((model) => {
    const matchesQuery = [model.name, model.slug, model.provider, model.modality, model.profile_name]
      .join(" ")
      .toLowerCase()
      .includes(query);
    const matchesModality = selectedModality === "all" || model.modality === selectedModality;
    return matchesQuery && matchesModality;
  });

  $("catalogMetrics").innerHTML = [
    `<div><strong>${models.length}</strong><small>Total models</small></div>`,
    `<div><strong>${new Set(models.map((model) => model.provider)).size}</strong><small>Providers</small></div>`,
    `<div><strong>${new Set(models.map((model) => model.modality)).size}</strong><small>Modalities</small></div>`
  ].join("");

  $("modelsGrid").innerHTML =
    visible
      .map((model) => {
        const price = priceFor(model);
        return `<article class="model-card">
          <button class="model-card-button" type="button" data-model-slug="${escapeHTML(model.slug)}" aria-label="Open ${escapeHTML(model.name)} details">
            <div class="model-art"></div>
          </button>
          <div class="model-card-body">
            <div class="model-card-head">
              <div>
                <h3>${escapeHTML(model.name)}</h3>
                <small class="provider-name">${escapeHTML(model.provider)}</small>
              </div>
              <span class="tag">${escapeHTML(model.modality)}</span>
            </div>
            <p>${escapeHTML(model.profile_name || "Default profile")} supports marketplace routing and credit metering.</p>
            <div class="tag-row">
              <span class="tag">${escapeHTML(price)}</span>
              <span class="tag">${model.profile_slug ? "Profile" : "Mock"}</span>
            </div>
            <button class="secondary-link model-detail-link" type="button" data-model-slug="${escapeHTML(model.slug)}">View Details</button>
          </div>
        </article>`;
      })
      .join("") || "<div class=\"empty-state\">No models match the current filters.</div>";
}

function showModelDetail(slug: string) {
  const model = models.find((item) => item.slug === slug);
  if (!model) return;
  $("modelDetailContent").innerHTML = renderModelDetail(model);
  setActiveTab("model-detail");
}

function renderModelDetail(model: Model) {
  const price = priceFor(model);
  const providerSlug = model.slug.includes("/") ? model.slug.split("/")[0] : model.provider.toLowerCase().replaceAll(" ", "-");
  const modelName = model.slug.includes("/") ? model.slug.split("/").slice(1).join("/") : model.slug;
  return `<div class="detail-hero">
    <div>
      <p class="eyebrow">${escapeHTML(model.provider)}</p>
      <h2>${escapeHTML(model.name)}</h2>
      <p class="detail-slug">${escapeHTML(providerSlug)} / ${escapeHTML(modelName)}</p>
      <p>${escapeHTML(descriptionFor(model))}</p>
      <div class="detail-actions">
        <button type="button" data-tab="workbench">Playground</button>
        <button class="ghost" type="button" data-copy-model="${escapeHTML(model.profile_slug || model.slug)}">Copy model ID</button>
      </div>
    </div>
    <aside class="detail-stats">
      <div><strong>${escapeHTML(model.modality)}</strong><small>Modality</small></div>
      <div><strong>${escapeHTML(price)}</strong><small>Price</small></div>
      <div><strong>${escapeHTML(model.status || "public")}</strong><small>Status</small></div>
    </aside>
  </div>
  <div class="detail-tabs" aria-label="Model detail sections">
    <span>Overview</span>
    <span>Providers</span>
    <span>Performance</span>
    <span>Benchmarks</span>
    <span>Activity</span>
    <span>Uptime</span>
    <span>API</span>
  </div>
  <div class="detail-grid">
    <article class="detail-panel">
      <h3>Overview</h3>
      <p>${escapeHTML(model.profile_name || "Default profile")} is available through the local marketplace catalog and can be routed through the workbench/API surface.</p>
      <div class="tag-row">${capabilityTags(model).map((tag) => `<span class="tag">${escapeHTML(tag)}</span>`).join("")}</div>
    </article>
    <article class="detail-panel">
      <h3>Provider routing</h3>
      <p>Requests route through ${escapeHTML(model.provider)} metadata. Fallback routing, uptime scoring, and provider comparisons are mocked for this detail page.</p>
      <div class="meta-row"><span><strong>Provider</strong><small>Catalog source</small></span><span>${escapeHTML(model.provider)}</span></div>
      <div class="meta-row"><span><strong>Profile</strong><small>Default configuration</small></span><span>${escapeHTML(model.profile_slug || "default")}</span></div>
    </article>
    <article class="detail-panel detail-api">
      <h3>API</h3>
      <pre>{
  "model": "${escapeHTML(model.profile_slug || model.slug)}",
  "messages": [
    { "role": "user", "content": "Generate a result" }
  ]
}</pre>
    </article>
  </div>`;
}

function descriptionFor(model: Model) {
  if (model.slug === "x-ai/grok-imagine-video") {
    return "Grok Imagine Video is xAI's fast text-, image-, and reference-conditioned video generation model for short clips across common aspect ratios.";
  }
  if (model.modality === "video") return `${model.name} generates video from prompt and reference inputs through the marketplace routing layer.`;
  if (model.modality === "image") return `${model.name} generates images from text and image prompts through the marketplace routing layer.`;
  if (model.modality === "audio") return `${model.name} supports audio generation or audio-aware workflows through the marketplace routing layer.`;
  return `${model.name} is available in the marketplace catalog.`;
}

function capabilityTags(model: Model) {
  if (model.modality === "video") return ["Text to video", "Image input", "Async job", "Provider routed"];
  if (model.modality === "image") return ["Text to image", "Image input", "Credit metered", "Provider routed"];
  if (model.modality === "audio") return ["Audio output", "Prompt input", "Credit metered", "Provider routed"];
  return ["Chat", "Streaming", "Credit metered"];
}

function priceFor(model: Model): string {
  if (model.modality === "image") return "10 credits / image";
  if (model.modality === "video") return "60 credits / 10 sec";
  if (model.modality === "audio") return "4 credits / minute";
  return "0.002 credits / 1k output";
}

async function loadAPIKeys() {
  try {
    const data = await request<{ api_keys: APIKey[] }>(`/api/v1/api-keys?project_id=${encodeURIComponent(currentProjectID)}`);
    $("apiKeys").innerHTML =
      data.api_keys
        .map((key) => `<div><strong>${escapeHTML(key.name)}</strong><small>${escapeHTML(key.prefix)}... ${escapeHTML(key.status)}</small></div>`)
        .join("") || "<div class=\"empty-state\">No API keys yet.</div>";
  } catch (error) {
    $("apiKeys").innerHTML = `<div class="empty-state">${escapeHTML(error instanceof Error ? error.message : String(error))}</div>`;
  }
}

async function createKey() {
  if (!currentProjectID) throw new Error("No project loaded. Run scripts/populate_test_data.sh dev first.");
  const data = await request<{ api_key: string; prefix: string }>("/api/v1/api-keys", {
    method: "POST",
    body: JSON.stringify({ project_id: currentProjectID, name: "Frontend dev key" })
  });
  currentAPIKey = data.api_key;
  $("newKey").innerHTML = `<p><strong>New key:</strong> <code>${escapeHTML(data.api_key)}</code></p>`;
  await loadAPIKeys();
}

async function sendPrompt() {
  if (!currentAPIKey) await createKey();
  const model = selectedWorkbenchModel || ($("modelSelect") as HTMLSelectElement).value || "mock-chat";
  const prompt = ($("prompt") as HTMLTextAreaElement).value;
  appendMessage("user", prompt);
  const data = await request<ChatResponse>("/api/v1/chat/completions", {
    method: "POST",
    headers: { Authorization: `Bearer ${currentAPIKey}` },
    body: JSON.stringify({ model, messages: [{ role: "user", content: prompt }] })
  });
  const answer = data.choices?.[0]?.message?.content || "Mock response received from backend.";
  appendMessage("assistant", answer);
  $("chatResult").textContent = JSON.stringify(data, null, 2);
  await loadSummary();
}

function appendMessage(role: "user" | "assistant", content: string) {
  const transcript = $("chatTranscript");
  transcript.insertAdjacentHTML("beforeend", `<div class="message ${role}">${escapeHTML(content)}</div>`);
  transcript.scrollTop = transcript.scrollHeight;
}

function renderMetadata() {
  const providerCount = new Set(models.map((model) => model.provider)).size;
  $("metadataRows").innerHTML = [
    metaRow("Providers", String(providerCount), "Backend catalog plus mock UI families"),
    metaRow("Model profiles", String(models.filter((model) => model.profile_slug).length), "System prompts and default parameters"),
    metaRow("Routing policies", "1", "Mock fixed-provider routing"),
    metaRow("Admin state", "Draft", "Publish and rollback planned for Phase 2")
  ].join("");
}

function metaRow(label: string, value: string, note: string) {
  return `<div class="meta-row"><span><strong>${escapeHTML(label)}</strong><small>${escapeHTML(note)}</small></span><span>${escapeHTML(value)}</span></div>`;
}

function renderPricing() {
  $("pricingRows").innerHTML = models
    .slice(0, 5)
    .map((model) => `<div class="price-row">
      <span><strong>${escapeHTML(model.name)}</strong><small>${escapeHTML(model.provider)}</small></span>
      <span>${escapeHTML(priceFor(model))}</span>
    </div>`)
    .join("");
}

function showError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  $("chatResult").textContent = message;
  appendMessage("assistant", message);
}

function setActiveTab(tab: string) {
  const nextTab = tabs.has(tab) ? tab : "home";
  document.querySelectorAll<HTMLElement>("[data-panel]").forEach((panel) => {
    panel.classList.toggle("active", panel.dataset.panel === nextTab);
  });
  document.querySelectorAll<HTMLButtonElement>("[data-tab]").forEach((button) => {
    button.classList.toggle("active", button.dataset.tab === nextTab);
  });
  if (window.location.hash.slice(1) !== nextTab) {
    window.history.replaceState(null, "", `#${nextTab}`);
  }
}

function openLogin() {
  $("loginModal").classList.remove("hidden");
  closeAccountMenu();
  $("authMessage").textContent = "";
  $("authUsername").focus();
}

function closeLogin() {
  $("loginModal").classList.add("hidden");
}

function setSignupMode(enabled: boolean) {
  signupMode = enabled;
  $("nameField").classList.toggle("hidden", !enabled);
  $("loginModeLabel").textContent = enabled ? "Create account" : "Account";
  $("loginTitle").textContent = enabled ? "Sign up for Model Market" : "Log in to Model Market";
  $("authSubmit").textContent = enabled ? "Create account" : "Login";
  $("authSwitchText").textContent = enabled ? "Already have an account?" : "No account yet?";
  $("toggleSignup").textContent = enabled ? "Login" : "Sign up";
  $("authMessage").textContent = "";
}

async function submitAuth(event: SubmitEvent) {
  event.preventDefault();
  const username = ($("authUsername") as HTMLInputElement).value.trim();
  const password = ($("authPassword") as HTMLInputElement).value;
  const name = ($("authName") as HTMLInputElement).value.trim();
  const path = signupMode ? "/api/v1/auth/signup" : "/api/v1/auth/login";
  const body = signupMode ? { email: username, name, password } : { username, password };
  $("authMessage").textContent = signupMode ? "Creating account..." : "Logging in...";
  try {
    const auth = await request<AuthResponse>(path, {
      method: "POST",
      body: JSON.stringify(body)
    });
    completeAuth(auth, signupMode ? "Account created" : "Logged in");
  } catch (error) {
    $("authMessage").textContent = error instanceof Error ? error.message : String(error);
  }
}

async function socialLogin(provider: string) {
  $("authMessage").textContent = `Logging in with ${provider}...`;
  try {
    const auth = await request<AuthResponse>("/api/v1/auth/social/dev", {
      method: "POST",
      body: JSON.stringify({ provider })
    });
    completeAuth(auth, `Logged in with ${provider}`);
  } catch (error) {
    $("authMessage").textContent = error instanceof Error ? error.message : String(error);
  }
}

function completeAuth(auth: AuthResponse, message: string) {
  if (!auth.user) {
    $("authMessage").textContent = "Login response did not include a user.";
    return;
  }
  if (auth.project_id) currentProjectID = auth.project_id;
  localStorage.setItem("authUser", JSON.stringify(auth.user));
  localStorage.setItem("authSession", JSON.stringify(auth.session || {}));
  renderAuthUser();
  $("authMessage").textContent = `${message} as ${auth.user.name}`;
  if (isAdminUser(auth.user)) {
    $("adminUserName").textContent = auth.user.name || auth.user.email;
    setActiveTab("admin");
  }
  window.setTimeout(closeLogin, 550);
}

function renderAuthUser() {
  const raw = localStorage.getItem("authUser");
  if (!raw) {
    $("loginButton").textContent = "Login";
    $("loginButton").setAttribute("aria-expanded", "false");
    closeAccountMenu();
    return;
  }
  try {
    const user = JSON.parse(raw) as { name?: string; email?: string };
    $("loginButton").textContent = user.name || user.email || "Account";
    if (isAdminUser(user)) {
      $("adminUserName").textContent = user.name || user.email || "Admin User";
    }
  } catch {
    localStorage.removeItem("authUser");
    localStorage.removeItem("authSession");
    renderAuthUser();
  }
}

function isAdminUser(user: { id?: string; email?: string }) {
  return user.id === "user-admin" || user.email === "admin@example.com";
}

function isSignedIn() {
  return Boolean(localStorage.getItem("authUser"));
}

function toggleAccountMenu() {
  if (!isSignedIn()) {
    openLogin();
    return;
  }
  const menu = $("accountMenu");
  const expanded = menu.classList.toggle("hidden") ? "false" : "true";
  $("loginButton").setAttribute("aria-expanded", expanded);
}

function closeAccountMenu() {
  $("accountMenu").classList.add("hidden");
  $("loginButton").setAttribute("aria-expanded", "false");
}

function accountAction(action: string) {
  closeAccountMenu();
  if (action === "logout") {
    localStorage.removeItem("authUser");
    localStorage.removeItem("authSession");
    renderAuthUser();
    setActiveTab("home");
    return;
  }
  window.history.replaceState(null, "", `#${action}`);
}

function applyTheme(mode: string) {
  const selected = themeModes.has(mode) ? mode : "system";
  const resolved = selected === "system" ? (systemTheme.matches ? "dark" : "light") : selected;
  document.documentElement.dataset.theme = resolved;
  document.documentElement.dataset.themeMode = selected;
  $("themeIcon").textContent = selected === "dark" ? "☾" : selected === "light" ? "☀" : "◐";
  ($("themeMode") as HTMLSelectElement).value = selected;
  localStorage.setItem("themeMode", selected);
}

function applyLanguage(language: string) {
  const selected = language === "zh" ? "zh" : "en";
  document.documentElement.lang = selected;
  ($("languageSelect") as HTMLSelectElement).value = selected;
  localStorage.setItem("language", selected);
}

$("loginButton").addEventListener("click", toggleAccountMenu);
$("accountMenu").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-account-action]");
  if (button) accountAction(button.dataset.accountAction || "");
});
document.addEventListener("click", (event) => {
  const target = event.target as Node;
  if (!$("accountMenu").contains(target) && !$("loginButton").contains(target)) closeAccountMenu();
});
$("closeLogin").addEventListener("click", closeLogin);
$("loginModal").addEventListener("click", (event) => {
  if (event.target === $("loginModal")) closeLogin();
});
$("toggleSignup").addEventListener("click", () => setSignupMode(!signupMode));
$("loginForm").addEventListener("submit", (event) => submitAuth(event as SubmitEvent));
document.querySelectorAll<HTMLButtonElement>(".social-button").forEach((button) => {
  button.addEventListener("click", () => socialLogin(button.dataset.provider || ""));
});
$("createKey").addEventListener("click", () => createKey().catch(showError));
$("createProject").addEventListener("click", () => createProject().catch(showError));
$("createConversation").addEventListener("click", () => createConversation().catch(showError));
$("projects").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-project-id]");
  if (!button) return;
  currentProjectID = button.dataset.projectId || currentProjectID;
  renderProjects();
  loadAPIKeys().catch(showError);
  loadWorkspace().catch(showError);
});
$("sendPrompt").addEventListener("click", () => sendPrompt().catch(showError));
$("backToModels").addEventListener("click", () => setActiveTab("models"));
$("modelsGrid").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-model-slug]");
  if (button) showModelDetail(button.dataset.modelSlug || "");
});
$("modelDetailContent").addEventListener("click", async (event) => {
  const tabButton = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-tab]");
  if (tabButton) setActiveTab(tabButton.dataset.tab || "workbench");
  const copyButton = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-copy-model]");
  if (copyButton) await navigator.clipboard?.writeText(copyButton.dataset.copyModel || "");
});
$("modelSearch").addEventListener("input", renderModels);
$("workbenchModelFilter").addEventListener("input", renderWorkbenchModelPicker);
document.querySelectorAll<HTMLButtonElement>("[data-workbench-modality]").forEach((button) => {
  button.addEventListener("click", () => {
    workbenchModality = button.dataset.workbenchModality || "image";
    const next = models.find((model) => model.modality === workbenchModality);
    if (next) selectedWorkbenchModel = next.profile_slug || next.slug;
    renderWorkbenchModelPicker();
  });
});
$("workbenchModelList").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-workbench-model]");
  if (!button) return;
  selectedWorkbenchModel = button.dataset.workbenchModel || selectedWorkbenchModel;
  renderWorkbenchModelPicker();
});
document.querySelectorAll<HTMLButtonElement>("[data-modality]").forEach((button) => {
  button.addEventListener("click", () => {
    selectedModality = button.dataset.modality || "all";
    document.querySelectorAll<HTMLButtonElement>("[data-modality]").forEach((item) => {
      item.classList.toggle("active", item.dataset.modality === selectedModality);
    });
    renderModels();
  });
});
$("themeMode").addEventListener("change", (event) => applyTheme((event.target as HTMLSelectElement).value));
$("languageSelect").addEventListener("change", (event) => applyLanguage((event.target as HTMLSelectElement).value));
systemTheme.addEventListener("change", () => {
  if (document.documentElement.dataset.themeMode === "system") applyTheme("system");
});
document.querySelectorAll<HTMLButtonElement>("[data-tab]").forEach((button) => {
  button.addEventListener("click", () => setActiveTab(button.dataset.tab || "models"));
});
window.addEventListener("hashchange", () => setActiveTab(window.location.hash.slice(1)));

applyTheme(localStorage.getItem("themeMode") || "system");
applyLanguage(localStorage.getItem("language") || "en");
renderAuthUser();
setActiveTab(window.location.hash.slice(1) || "home");
loadAll().catch(showError);
