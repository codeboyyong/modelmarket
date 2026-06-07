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

type Model = {
  slug: string;
  name: string;
  provider: string;
  modality: string;
  profile_slug: string;
  profile_name: string;
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
let signupMode = false;
let selectedModality = "all";
const tabs = new Set(["home", "models", "api", "workbench", "admin", "pricing"]);
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
  if (currentProjectID) await loadAPIKeys();
}

function renderProjects() {
  $("projects").innerHTML =
    projects
      .map((project) => {
        const credits = project.paid_credits + project.promotional_credits;
        return `<div>
          <strong>${escapeHTML(project.name)}</strong>
          <small>${escapeHTML(project.organization)}</small>
          <p>${credits.toLocaleString()} credits available</p>
        </div>`;
      })
      .join("") || "<div class=\"empty-state\">No project loaded. Run the dev test data script first.</div>";
}

async function loadModels() {
  try {
    const data = await request<{ models: Model[] }>("/api/v1/models");
    models = [...data.models, ...mockModels];
  } catch {
    models = [...mockModels];
  }
  renderModelControls();
  renderModels();
  renderMetadata();
  renderPricing();
}

function renderModelControls() {
  const chatModels = models.filter((model) => model.modality === "chat");
  $("modelSelect").innerHTML =
    chatModels
      .map((model) => `<option value="${escapeHTML(model.profile_slug || model.slug)}">${escapeHTML(model.name)}</option>`)
      .join("") || "<option value=\"mock-chat\">Mock Chat</option>";
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
          <div class="model-art"></div>
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
          </div>
        </article>`;
      })
      .join("") || "<div class=\"empty-state\">No models match the current filters.</div>";
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
  const model = ($("modelSelect") as HTMLSelectElement).value || "mock-chat";
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
$("sendPrompt").addEventListener("click", () => sendPrompt().catch(showError));
$("modelSearch").addEventListener("input", renderModels);
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
