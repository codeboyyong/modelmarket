type Summary = {
  counts: Record<string, number>;
};

type Project = {
  id: string;
  name: string;
  organization: string;
  paid_credits: number;
  promotional_credits: number;
  credits_used: number;
};

type Conversation = {
  id: string;
  title: string;
  status: string;
  message_count: number;
};

type ConversationBranch = {
  id: string;
  conversation_id: string;
  parent_branch_id: string;
  name: string;
  message_count: number;
  created_at: string;
};

type WorkspaceAsset = {
  id: string;
  conversation_id: string;
  asset_type: string;
  storage_path: string;
  storage_provider: string;
  bucket_name: string;
  object_key: string;
  download_url: string;
  mime_type: string;
  size_bytes: number;
  customer_charge: number;
  provider_cost: number;
};

type UploadIntentResponse = {
  asset: WorkspaceAsset;
  message_id: string;
  upload: {
    method: string;
    url: string;
  };
};

type ConversationMessage = {
  id: string;
  role: "user" | "assistant" | string;
  content: string;
  model_profile_id: string;
  inference_request_id: string;
  customer_charge: number;
  provider_cost: number;
  metadata: string;
  created_at: string;
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

type PricingRow = {
  provider: string;
  model: string;
  model_slug: string;
  modality: string;
  profile: string;
  profile_slug: string;
  input_price: number;
  input_price_unit: string;
  output_price: number;
  output_price_unit: string;
  currency: string;
};

type CreditRange = "7" | "30" | "90";

type UsageSlice = {
  type: "image" | "audio" | "video";
  label: string;
  credits: number;
  color: string;
};

type UserTokenUsageRow = {
  date: string;
  model: string;
  model_slug: string;
  modality?: string;
  input_tokens: number;
  output_tokens: number;
  customer_charge?: number;
  color: string;
};

type CreditPurchase = {
  date: string;
  credits: number;
  amount?: string;
  amount_cents?: number;
  currency?: string;
  status: string;
};

type UserCreditUsageResponse = {
  user_id: string;
  range_days: number;
  total_tokens: number;
  credits_bought: number;
  top_model: string;
  usage: Array<Omit<UserTokenUsageRow, "color">>;
  purchases: CreditPurchase[];
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
    user_type?: string;
    company_id?: string;
    company_name?: string;
  };
  project_id?: string;
  provider?: string;
  session?: {
    access_token: string;
    token_type: string;
  };
};

type AuthUser = NonNullable<AuthResponse["user"]>;

type CompanyUsageMember = {
  id: string;
  email: string;
  name: string;
  user_type: string;
  credits_used: number;
};

type CompanyUsageModel = {
  model_slug: string;
  model: string;
  modality: string;
  credits_used: number;
};

type CompanyUsageResponse = {
  company: {
    id: string;
    name: string;
  };
  total_credits: number;
  members: CompanyUsageMember[];
  models: CompanyUsageModel[];
};

const apiBase = (window as Window & { API_BASE_URL?: string }).API_BASE_URL || "http://localhost:8080";
let currentProjectID = "";
let currentAPIKey = "";
let models: Model[] = [];
let projects: Project[] = [];
let conversations: Conversation[] = [];
let branches: ConversationBranch[] = [];
let assets: WorkspaceAsset[] = [];
let pricingRows: PricingRow[] = [];
let signupMode = false;
let changePasswordMode = false;
let selectedModality = "all";
let selectedPricingModality = "all";
let workbenchModality = "image";
let selectedWorkbenchModel = "";
let workbenchPickerOpen = false;
let projectPickerOpen = false;
let conversationPickerOpen = false;
let currentConversationID = "";
let currentBranchID = "";
let branchSourceMessageID = "";
const tabs = new Set(["home", "models", "model-detail", "api", "workbench", "admin", "corporate-admin", "pricing", "credit-usage", "company", "privacy", "terms"]);
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
  await Promise.all([loadSummary(), loadProjects(), loadModels(), loadPricing()]);
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
  if (currentProjectID) await loadWorkspace();
}

function renderProjects() {
  const filter = ($("projectFilter") as HTMLInputElement).value.trim().toLowerCase();
  const visible = projects.filter((project) => [project.name, project.organization].join(" ").toLowerCase().includes(filter));
  $("projects").innerHTML =
    visible
      .map((project) => {
        const credits = project.paid_credits + project.promotional_credits;
        const active = project.id === currentProjectID ? " active" : "";
        return `<button class="workspace-row${active}" type="button" data-project-id="${escapeHTML(project.id)}">
          <span><strong>${escapeHTML(project.name)}</strong><small>${escapeHTML(project.organization)}</small></span>
          <p>${(project.credits_used || 0).toLocaleString()} credits used / ${credits.toLocaleString()} available</p>
        </button>`;
      })
      .join("") || "<div class=\"empty-state\">No project loaded. Run the dev test data script first.</div>";
    $("projects").classList.toggle("hidden", !projectPickerOpen);
    renderSelectedProject();
    renderCreditAnalytics();
}

function renderSelectedProject() {
  const selected = projects.find((project) => project.id === currentProjectID);
  if (!projectPickerOpen) ($("projectFilter") as HTMLInputElement).value = selected?.name || "";
}

function openProjectPicker(clearFilter = false) {
  projectPickerOpen = true;
  if (clearFilter) ($("projectFilter") as HTMLInputElement).value = "";
  renderProjects();
}

function closeProjectPicker() {
  projectPickerOpen = false;
  renderProjects();
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
  currentAPIKey = "";
  currentConversationID = "";
  currentBranchID = "";
  renderProjects();
  await loadWorkspace();
}

async function loadWorkspace() {
  if (!currentProjectID) {
    conversations = [];
    branches = [];
    assets = [];
    currentBranchID = "";
    renderWorkspaceLists();
    renderBranches();
    return;
  }
  try {
    const [conversationData, assetData] = await Promise.all([
      request<{ conversations: Conversation[] }>(`/api/v1/conversations?project_id=${encodeURIComponent(currentProjectID)}`),
      request<{ assets: WorkspaceAsset[] }>(`/api/v1/assets?project_id=${encodeURIComponent(currentProjectID)}`)
    ]);
    conversations = conversationData.conversations;
    assets = assetData.assets;
    if (!conversations.some((conversation) => conversation.id === currentConversationID)) {
      currentConversationID = conversations[0]?.id || "";
      currentBranchID = "";
    }
  } catch {
    conversations = [];
    branches = [];
    assets = [];
    currentConversationID = "";
    currentBranchID = "";
  }
  renderWorkspaceLists();
  await loadConversationBranches();
  await loadConversationMessages();
}

function renderWorkspaceLists() {
  const conversationFilter = ($("conversationFilter") as HTMLInputElement).value.trim().toLowerCase();
  const visibleConversations = conversations.filter((conversation) =>
    [conversation.title, conversation.status].join(" ").toLowerCase().includes(conversationFilter)
  );
  $("conversations").innerHTML =
    visibleConversations
      .map((conversation) => `<button class="workspace-row${conversation.id === currentConversationID ? " active" : ""}" type="button" data-conversation-id="${escapeHTML(conversation.id)}">
        <span><strong>${escapeHTML(conversation.title)}</strong><small>${conversation.message_count} messages / ${escapeHTML(conversation.status)}</small></span>
      </button>`)
      .join("") || "<div class=\"empty-state\">No conversations yet.</div>";
  $("conversations").classList.toggle("hidden", !conversationPickerOpen);
  $("artifacts").innerHTML =
    assets
      .map((asset) => `<div class="workspace-row artifact-row">
        <span class="artifact-transfer-icon ${artifactTransferClass(asset)}" title="${artifactTransferLabel(asset)}" aria-label="${artifactTransferLabel(asset)}">${artifactTransferIcon(asset)}</span>
        <span><strong>${escapeHTML(asset.asset_type)}</strong><small>${escapeHTML(asset.mime_type || "artifact")} / ${asset.size_bytes.toLocaleString()} bytes</small></span>
        <a class="artifact-link" href="${escapeHTML(asset.download_url || asset.storage_path)}" target="_blank" rel="noreferrer">${escapeHTML(asset.download_url || asset.storage_path)}</a>
      </div>`)
      .join("") || "<div class=\"empty-state\">No generated artifacts yet.</div>";
  renderSelectedConversation();
}

function artifactTransferLabel(asset: WorkspaceAsset) {
  return asset.asset_type === "upload" ? "Uploaded artifact" : "Downloadable artifact";
}

function artifactTransferClass(asset: WorkspaceAsset) {
  return asset.asset_type === "upload" ? "uploaded" : "downloaded";
}

function artifactTransferIcon(asset: WorkspaceAsset) {
  return asset.asset_type === "upload" ? "↑" : "↓";
}

function renderSelectedConversation() {
  const selected = conversations.find((conversation) => conversation.id === currentConversationID) || conversations[0];
  if (selected && !currentConversationID) currentConversationID = selected.id;
  if (!conversationPickerOpen) ($("conversationFilter") as HTMLInputElement).value = selected?.title || "";
}

async function loadConversationBranches() {
  if (!currentConversationID) {
    branches = [];
    currentBranchID = "";
    renderBranches();
    return;
  }
  try {
    const data = await request<{ branches: ConversationBranch[] }>(`/api/v1/conversation-branches?conversation_id=${encodeURIComponent(currentConversationID)}`);
    branches = data.branches;
    if (!branches.some((branch) => branch.id === currentBranchID)) {
      currentBranchID = branches[0]?.id || "";
    }
  } catch {
    branches = [];
    currentBranchID = "";
  }
  renderBranches();
}

function renderBranches() {
  const select = $("branchSelect") as HTMLSelectElement;
  $("branchPicker").classList.toggle("hidden", branches.length <= 1);
  select.disabled = branches.length === 0;
  select.innerHTML =
    branches
      .map((branch) => `<option value="${escapeHTML(branch.id)}"${branch.id === currentBranchID ? " selected" : ""}>${escapeHTML(branch.name)} (${branch.message_count})</option>`)
      .join("") || "<option value=\"\">Select a conversation first</option>";
}

function openConversationPicker(clearFilter = false) {
  conversationPickerOpen = true;
  if (clearFilter) ($("conversationFilter") as HTMLInputElement).value = "";
  renderWorkspaceLists();
}

function closeConversationPicker() {
  conversationPickerOpen = false;
  renderWorkspaceLists();
}

function closeFooterMenus() {
  document.querySelectorAll<HTMLElement>(".footer-menu.open").forEach((menu) => {
    menu.classList.remove("open");
  });
  document.querySelectorAll<HTMLButtonElement>("[data-footer-menu]").forEach((button) => {
    button.setAttribute("aria-expanded", "false");
  });
}

async function loadConversationMessages() {
  const transcript = $("chatTranscript");
  if (!currentConversationID) {
    transcript.innerHTML = "<div class=\"message assistant\">Select or create a conversation to resume work.</div>";
    return;
  }
  transcript.innerHTML = "<div class=\"message assistant\">Loading conversation...</div>";
  try {
    const branchQuery = currentBranchID ? `&branch_id=${encodeURIComponent(currentBranchID)}` : "";
    const data = await request<{ messages: ConversationMessage[] }>(`/api/v1/messages?conversation_id=${encodeURIComponent(currentConversationID)}${branchQuery}`);
    transcript.innerHTML =
      data.messages
        .map((message) => renderConversationMessage(message))
        .join("") || "<div class=\"message assistant\">This conversation is empty. Send a prompt to start.</div>";
    transcript.scrollTop = transcript.scrollHeight;
  } catch (error) {
    transcript.innerHTML = `<div class="message assistant">${escapeHTML(error instanceof Error ? error.message : String(error))}</div>`;
  }
}

function renderConversationMessage(message: ConversationMessage) {
  const role = message.role === "user" ? "user" : "assistant";
  return `<div class="message ${role}" data-message-id="${escapeHTML(message.id)}">${escapeHTML(message.content)}</div>`;
}

function openMessageContextMenu(messageID: string, x: number, y: number) {
  branchSourceMessageID = messageID;
  const menu = $("messageContextMenu");
  menu.style.left = `${x}px`;
  menu.style.top = `${y}px`;
  menu.classList.remove("hidden");
}

function closeMessageContextMenu() {
  branchSourceMessageID = "";
  $("messageContextMenu").classList.add("hidden");
}

async function startBranchFromMessage() {
  if (!currentConversationID || !branchSourceMessageID) return;
  const data = await request<{ branch: ConversationBranch }>("/api/v1/conversation-branches", {
    method: "POST",
    body: JSON.stringify({ conversation_id: currentConversationID, source_message_id: branchSourceMessageID })
  });
  currentBranchID = data.branch.id;
  closeMessageContextMenu();
  await loadConversationBranches();
  currentBranchID = data.branch.id;
  renderBranches();
  await loadConversationMessages();
}

async function createConversation() {
  if (!currentProjectID) throw new Error("Create or select a project first.");
  const title = window.prompt("Conversation title") || "New conversation";
  const data = await request<{ conversation: Conversation }>("/api/v1/conversations", {
    method: "POST",
    body: JSON.stringify({ project_id: currentProjectID, title })
  });
  conversations = [data.conversation, ...conversations];
  currentConversationID = data.conversation.id;
  currentBranchID = "";
  renderWorkspaceLists();
  await loadConversationBranches();
  await loadConversationMessages();
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
  renderSelectedWorkbenchModel();
  renderWorkbenchModelPicker();
}

function renderWorkbenchModelPicker() {
  document.querySelectorAll<HTMLButtonElement>("[data-workbench-modality]").forEach((button) => {
    button.classList.toggle("active", button.dataset.workbenchModality === workbenchModality);
  });
  const filterInput = $("workbenchModelFilter") as HTMLInputElement;
  const query = filterInput.value.trim().toLowerCase();
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
  $("workbenchModelList").classList.toggle("hidden", !workbenchPickerOpen);
}

function renderSelectedWorkbenchModel() {
  const selected = models.find((model) => (model.profile_slug || model.slug) === selectedWorkbenchModel);
  $("selectedModelName").textContent = selected?.name || "No model selected";
  $("selectedModelModality").textContent = selected?.modality || "-";
  ($("modelSelect") as HTMLSelectElement).value = selectedWorkbenchModel;
  const input = $("workbenchModelFilter") as HTMLInputElement;
  if (!workbenchPickerOpen) input.value = selected?.name || "";
}

function openWorkbenchPicker(clearFilter = false) {
  workbenchPickerOpen = true;
  if (clearFilter) ($("workbenchModelFilter") as HTMLInputElement).value = "";
  renderWorkbenchModelPicker();
}

function closeWorkbenchPicker() {
  workbenchPickerOpen = false;
  renderSelectedWorkbenchModel();
  renderWorkbenchModelPicker();
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
    currentAPIKey = "";
    void data;
  } catch (error) {
    currentAPIKey = "";
    void error;
  }
}

async function createKey() {
  if (!currentProjectID) throw new Error("No project loaded. Run scripts/populate_test_data.sh dev first.");
  const data = await request<{ api_key: string; prefix: string }>("/api/v1/api-keys", {
    method: "POST",
    body: JSON.stringify({ project_id: currentProjectID, name: "Frontend dev key" })
  });
  currentAPIKey = data.api_key;
}

async function sendPrompt() {
  if (!currentAPIKey) await createKey();
  if (!currentConversationID) throw new Error("Select or create a conversation first.");
  const model = selectedWorkbenchModel || ($("modelSelect") as HTMLSelectElement).value || "mock-chat";
  const prompt = ($("prompt") as HTMLTextAreaElement).value;
  appendMessage("user", prompt);
  const data = await request<ChatResponse>("/api/v1/chat/completions", {
    method: "POST",
    headers: { Authorization: `Bearer ${currentAPIKey}` },
    body: JSON.stringify({ model, conversation_id: currentConversationID, branch_id: currentBranchID, messages: [{ role: "user", content: prompt }] })
  });
  const answer = data.choices?.[0]?.message?.content || "Mock response received from backend.";
  appendMessage("assistant", answer);
  await loadSummary();
  await loadWorkspace();
}

async function uploadAssetFile(file: File) {
  if (!currentProjectID) throw new Error("Select or create a project first.");
  const data = await request<UploadIntentResponse>("/api/v1/assets/upload-intent", {
    method: "POST",
    body: JSON.stringify({
      project_id: currentProjectID,
      conversation_id: currentConversationID,
      branch_id: currentBranchID,
      filename: file.name,
      content_type: file.type,
      size_bytes: file.size
    })
  });
  appendMessage("user", `Uploaded ${file.name}`);
  void data;
  await loadWorkspace();
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
  const query = ($("pricingSearch") as HTMLInputElement).value.trim().toLowerCase();
  const visible = pricingRows.filter((row) => {
    const matchesQuery = [row.provider, row.model, row.model_slug, row.profile].join(" ").toLowerCase().includes(query);
    const matchesModality = selectedPricingModality === "all" || row.modality === selectedPricingModality;
    return matchesQuery && matchesModality;
  });
  $("pricingRows").innerHTML =
    visible
      .map((row) => `<tr>
        <td><strong>${escapeHTML(row.provider)}</strong><small>${escapeHTML(row.modality)}</small></td>
        <td>
          <button class="table-link" type="button" data-pricing-model-slug="${escapeHTML(row.model_slug)}">${escapeHTML(row.model)}</button>
          <small>${escapeHTML(row.model_slug)}</small>
        </td>
        <td>${escapeHTML(formatPrice(row.input_price, row.input_price_unit, row.currency))}</td>
        <td>${escapeHTML(formatPrice(row.output_price, row.output_price_unit, row.currency))}</td>
      </tr>`)
      .join("") || "<tr><td colspan=\"4\">No pricing rows match the current filters.</td></tr>";
}

async function renderCreditAnalytics() {
  const range = (($("creditRange") as HTMLSelectElement).value || "30") as CreditRange;
  const user = getStoredUser();
  try {
    const path = `/api/v1/user-credit-usage?range=${encodeURIComponent(range)}${user?.id ? `&user_id=${encodeURIComponent(user.id)}` : ""}`;
    const data = await request<UserCreditUsageResponse>(path);
    const usageRows = data.usage.map((row) => ({ ...row, color: colorForModel(row.model_slug) }));
    renderCreditAnalyticsData(range, usageRows, data.purchases);
  } catch {
    const usageRows = buildUserTokenUsage(range);
    const purchases = buildUserCreditPurchases(range);
    renderCreditAnalyticsData(range, usageRows, purchases);
  }
}

function renderCreditAnalyticsData(range: CreditRange, usageRows: UserTokenUsageRow[], purchases: CreditPurchase[]) {
  const balancePoints = buildBalanceHistoryPoints(range, usageRows, purchases);
  const currentBalance = balancePoints[balancePoints.length - 1] || 0;
  $("currentCreditBalance").textContent = `${Math.round(currentBalance).toLocaleString()} credits`;
  renderCreditLineChart(balancePoints);
  renderCreditUsageChart(buildCreditUsageByType(usageRows));
}

function buildCreditUsageByType(rows: UserTokenUsageRow[]): UsageSlice[] {
  const totals = new Map<"image" | "audio" | "video", number>([
    ["image", 0],
    ["audio", 0],
    ["video", 0]
  ]);
  rows.forEach((row) => {
    const type = usageTypeForRow(row);
    totals.set(type, (totals.get(type) || 0) + chargeForUsageRow(row));
  });
  return [
    { type: "image", label: "Image", credits: totals.get("image") || 0, color: "#3f8cff" },
    { type: "audio", label: "Audio", credits: totals.get("audio") || 0, color: "#12b981" },
    { type: "video", label: "Video", credits: totals.get("video") || 0, color: "#f59e0b" }
  ];
}

function renderCreditLineChart(points: number[]) {
  const width = 320;
  const height = 132;
  const pad = 10;
  const min = Math.min(...points);
  const max = Math.max(...points);
  const span = Math.max(1, max - min);
  const coords = points.map((value, index) => {
    const x = pad + (index / Math.max(1, points.length - 1)) * (width - pad * 2);
    const y = pad + (1 - (value - min) / span) * (height - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const last = points[points.length - 1] || 0;
  const first = points[0] || last;
  const change = last - first;
  $("creditLineChart").innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" aria-hidden="true">
      <line x1="${pad}" y1="${height - pad}" x2="${width - pad}" y2="${height - pad}" />
      <line x1="${pad}" y1="${pad}" x2="${pad}" y2="${height - pad}" />
      <polyline points="${coords.join(" ")}" />
      <circle cx="${coords[coords.length - 1]?.split(",")[0] || pad}" cy="${coords[coords.length - 1]?.split(",")[1] || pad}" r="4" />
    </svg>
    <div class="chart-note"><span>${first.toLocaleString()} start</span><strong>${change >= 0 ? "+" : ""}${change.toLocaleString()}</strong></div>
  `;
}

function renderCreditUsageChart(slices: UsageSlice[]) {
  const total = slices.reduce((sum, slice) => sum + slice.credits, 0) || 1;
  let cursor = 0;
  const gradient = slices
    .map((slice) => {
      const start = cursor;
      cursor += (slice.credits / total) * 100;
      return `${slice.color} ${start.toFixed(2)}% ${cursor.toFixed(2)}%`;
    })
    .join(", ");
  $("creditPieChart").style.background = `conic-gradient(${gradient})`;
  $("creditPieChart").innerHTML = `<span>${total.toLocaleString()}<small>credits</small></span>`;
  $("creditUsageLegend").innerHTML = slices
    .map((slice) => {
      const percent = Math.round((slice.credits / total) * 100);
      return `<div><i style="background:${slice.color}"></i><span>${escapeHTML(slice.label)}</span><strong>${percent}%</strong></div>`;
    })
    .join("");
}

async function renderUserCreditUsage() {
  const range = (($("userUsageRange") as HTMLSelectElement).value || "30") as CreditRange;
  const user = getStoredUser();
  try {
    const path = `/api/v1/user-credit-usage?range=${encodeURIComponent(range)}${user?.id ? `&user_id=${encodeURIComponent(user.id)}` : ""}`;
    const data = await request<UserCreditUsageResponse>(path);
    const usageRows = data.usage.map((row) => ({ ...row, color: colorForModel(row.model_slug) }));
    renderUserCreditUsageData(range, usageRows, data.purchases, data.total_tokens, data.credits_bought, data.top_model || topUserUsageModel(usageRows));
  } catch {
    const usageRows = buildUserTokenUsage(range);
    const purchases = buildUserCreditPurchases(range);
    const totalTokens = usageRows.reduce((sum, row) => sum + row.input_tokens + row.output_tokens, 0);
    const totalCreditsBought = purchases.reduce((sum, purchase) => sum + purchase.credits, 0);
    renderUserCreditUsageData(range, usageRows, purchases, totalTokens, totalCreditsBought, topUserUsageModel(usageRows));
  }
}

function renderUserCreditUsageData(range: CreditRange, usageRows: UserTokenUsageRow[], purchases: CreditPurchase[], totalTokens: number, totalCreditsBought: number, topModel: string) {
  $("userTokenTotal").textContent = totalTokens.toLocaleString();
  $("userCreditBought").textContent = totalCreditsBought.toLocaleString();
  $("userTopModel").textContent = topModel;
  renderUserTokenLineChart(usageRows);
  renderUserBalanceLineChart(range, usageRows, purchases);
  renderUserModelPieChart(usageRows);
  renderUserDailyUsageTable(usageRows);
  renderUserPurchaseTable(purchases);
}

function buildUserTokenUsage(range: CreditRange): UserTokenUsageRow[] {
  const days = Number(range);
  const modelSeed = [
    { model: "xAI: Grok Imagine Video", model_slug: "x-ai/grok-imagine-video", inputBase: 820, outputBase: 1640, color: "#f59e0b" },
    { model: "Microsoft: MAI-Image-2.5", model_slug: "microsoft/mai-image-2.5", inputBase: 520, outputBase: 980, color: "#3f8cff" },
    { model: "OpenAI: GPT Audio", model_slug: "openai/gpt-audio", inputBase: 430, outputBase: 760, color: "#12b981" }
  ];
  return Array.from({ length: days }, (_, dayIndex) => {
    const date = dateDaysAgo(days - dayIndex - 1);
    return modelSeed.map((model, modelIndex) => {
      const wave = 1 + Math.sin((dayIndex + 1) * (modelIndex + 2) * 0.42) * 0.18;
      const weekdayBoost = dayIndex % 7 === 2 || dayIndex % 7 === 3 ? 1.22 : 1;
      const rangeBoost = days === 7 ? 1.12 : days === 90 ? 0.86 : 1;
      return {
        date,
        model: model.model,
        model_slug: model.model_slug,
        input_tokens: Math.round(model.inputBase * wave * weekdayBoost * rangeBoost),
        output_tokens: Math.round(model.outputBase * wave * weekdayBoost * rangeBoost),
        color: model.color
      };
    });
  }).flat();
}

function buildUserCreditPurchases(range: CreditRange): CreditPurchase[] {
  const purchases = [
    { date: dateDaysAgo(3), credits: 5000, amount: "$50.00", status: "Posted" },
    { date: dateDaysAgo(18), credits: 10000, amount: "$100.00", status: "Posted" },
    { date: dateDaysAgo(44), credits: 5000, amount: "$50.00", status: "Posted" },
    { date: dateDaysAgo(72), credits: 20000, amount: "$200.00", status: "Posted" }
  ];
  return purchases.filter((purchase) => daysBetween(purchase.date) < Number(range));
}

function renderUserDailyUsageTable(rows: UserTokenUsageRow[]) {
  $("userDailyUsageRows").innerHTML = rows
    .slice()
    .reverse()
    .map((row) => {
      const total = row.input_tokens + row.output_tokens;
      return `<tr>
        <td>${escapeHTML(row.date)}</td>
        <td><strong>${escapeHTML(row.model)}</strong><small>${escapeHTML(row.model_slug)}</small></td>
        <td>${row.input_tokens.toLocaleString()}</td>
        <td>${row.output_tokens.toLocaleString()}</td>
        <td>${total.toLocaleString()}</td>
      </tr>`;
    })
    .join("");
}

function renderUserPurchaseTable(purchases: CreditPurchase[]) {
  $("userCreditPurchaseRows").innerHTML =
    purchases
      .map((purchase) => `<tr>
        <td>${escapeHTML(purchase.date)}</td>
        <td>${purchase.credits.toLocaleString()}</td>
        <td>${escapeHTML(formatPurchaseAmount(purchase))}</td>
        <td>${escapeHTML(purchase.status)}</td>
      </tr>`)
      .join("") || "<tr><td colspan=\"4\">No credit purchases in this range.</td></tr>";
}

function renderUserTokenLineChart(rows: UserTokenUsageRow[]) {
  const dailyTotals = new Map<string, number>();
  rows.forEach((row) => {
    dailyTotals.set(row.date, (dailyTotals.get(row.date) || 0) + row.input_tokens + row.output_tokens);
  });
  renderLineChart("userTokenLineChart", Array.from(dailyTotals.values()), "tokens");
}

function renderUserBalanceLineChart(range: CreditRange, usageRows: UserTokenUsageRow[], purchases: CreditPurchase[]) {
  renderLineChart("userBalanceLineChart", buildBalanceHistoryPoints(range, usageRows, purchases), "credits");
}

function buildBalanceHistoryPoints(range: CreditRange, usageRows: UserTokenUsageRow[], purchases: CreditPurchase[]) {
  const days = Number(range);
  let balance = 12000;
  const usageByDate = new Map<string, number>();
  usageRows.forEach((row) => {
    usageByDate.set(row.date, (usageByDate.get(row.date) || 0) + chargeForUsageRow(row));
  });
  const purchasesByDate = new Map<string, number>();
  purchases.forEach((purchase) => {
    purchasesByDate.set(purchase.date, (purchasesByDate.get(purchase.date) || 0) + purchase.credits);
  });
  const points = Array.from({ length: days }, (_, index) => {
    const date = dateDaysAgo(days - index - 1);
    balance += purchasesByDate.get(date) || 0;
    balance -= usageByDate.get(date) || 0;
    return Math.max(0, balance);
  });
  return points;
}

function renderUserModelPieChart(rows: UserTokenUsageRow[]) {
  const byModel = new Map<string, { total: number; color: string }>();
  rows.forEach((row) => {
    const current = byModel.get(row.model) || { total: 0, color: row.color };
    current.total += row.input_tokens + row.output_tokens;
    byModel.set(row.model, current);
  });
  const slices = Array.from(byModel.entries()).map(([label, value]) => ({ label, credits: value.total, color: value.color }));
  const total = slices.reduce((sum, slice) => sum + slice.credits, 0) || 1;
  let cursor = 0;
  const gradient = slices
    .map((slice) => {
      const start = cursor;
      cursor += (slice.credits / total) * 100;
      return `${slice.color} ${start.toFixed(2)}% ${cursor.toFixed(2)}%`;
    })
    .join(", ");
  $("userModelPieChart").style.background = `conic-gradient(${gradient})`;
  $("userModelPieChart").innerHTML = `<span>${total.toLocaleString()}<small>tokens</small></span>`;
  $("userModelUsageLegend").innerHTML = slices
    .map((slice) => {
      const percent = Math.round((slice.credits / total) * 100);
      return `<div><i style="background:${slice.color}"></i><span>${escapeHTML(slice.label)}</span><strong>${percent}%</strong></div>`;
    })
    .join("");
}

function renderLineChart(elementID: string, points: number[], unit: string) {
  const width = 320;
  const height = 132;
  const pad = 10;
  const min = Math.min(...points);
  const max = Math.max(...points);
  const span = Math.max(1, max - min);
  const coords = points.map((value, index) => {
    const x = pad + (index / Math.max(1, points.length - 1)) * (width - pad * 2);
    const y = pad + (1 - (value - min) / span) * (height - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const last = points[points.length - 1] || 0;
  const first = points[0] || last;
  const change = last - first;
  $(elementID).innerHTML = `
    <svg viewBox="0 0 ${width} ${height}" aria-hidden="true">
      <line x1="${pad}" y1="${height - pad}" x2="${width - pad}" y2="${height - pad}" />
      <line x1="${pad}" y1="${pad}" x2="${pad}" y2="${height - pad}" />
      <polyline points="${coords.join(" ")}" />
      <circle cx="${coords[coords.length - 1]?.split(",")[0] || pad}" cy="${coords[coords.length - 1]?.split(",")[1] || pad}" r="4" />
    </svg>
    <div class="chart-note"><span>${first.toLocaleString()} start</span><strong>${change >= 0 ? "+" : ""}${change.toLocaleString()} ${escapeHTML(unit)}</strong></div>
  `;
}

function topUserUsageModel(rows: UserTokenUsageRow[]) {
  const totals = new Map<string, number>();
  rows.forEach((row) => totals.set(row.model, (totals.get(row.model) || 0) + row.input_tokens + row.output_tokens));
  const [top] = Array.from(totals.entries()).sort((a, b) => b[1] - a[1]);
  return top ? top[0] : "-";
}

function chargeForUsageRow(row: UserTokenUsageRow) {
  if (typeof row.customer_charge === "number") return row.customer_charge;
  return Math.max(1, Math.round((row.input_tokens + row.output_tokens) / 100));
}

function usageTypeForRow(row: UserTokenUsageRow): "image" | "audio" | "video" {
  const value = `${row.modality || ""} ${row.model_slug}`.toLowerCase();
  if (value.includes("audio")) return "audio";
  if (value.includes("video")) return "video";
  return "image";
}

function colorForModel(modelSlug: string) {
  if (modelSlug.includes("video")) return "#f59e0b";
  if (modelSlug.includes("audio")) return "#12b981";
  if (modelSlug.includes("image")) return "#3f8cff";
  return "#8b5cf6";
}

function formatPurchaseAmount(purchase: CreditPurchase) {
  if (purchase.amount) return purchase.amount;
  const amount = (purchase.amount_cents || 0) / 100;
  return `${purchase.currency || "USD"} ${amount.toFixed(2)}`;
}

function dateDaysAgo(daysAgo: number) {
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  date.setDate(date.getDate() - daysAgo);
  return date.toISOString().slice(0, 10);
}

function daysBetween(dateValue: string) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const then = new Date(`${dateValue}T00:00:00`);
  return Math.round((today.getTime() - then.getTime()) / 86400000);
}

async function loadCorporateUsage(user: AuthUser) {
  if (!isCorporateAdmin(user)) {
    renderCorporateUsage(null);
    return;
  }
  try {
    const data = await request<CompanyUsageResponse>(`/api/v1/company-usage?user_id=${encodeURIComponent(user.id)}`);
    renderCorporateUsage(data);
  } catch (error) {
    $("corporateMemberRows").innerHTML = `<tr><td colspan="4">${escapeHTML(error instanceof Error ? error.message : String(error))}</td></tr>`;
    $("corporateUsageLegend").innerHTML = "";
    $("corporateUsageChart").innerHTML = "<span>0<small>credits</small></span>";
  }
}

function renderCorporateUsage(data: CompanyUsageResponse | null) {
  if (!data) {
    $("corporateCompanyName").textContent = "Company usage";
    $("corporateTotalCredits").textContent = "0";
    $("corporateMemberCount").textContent = "0";
    $("corporateTopModality").textContent = "-";
    $("corporateMemberRows").innerHTML = "<tr><td colspan=\"4\">Login as a corporate admin to load company members.</td></tr>";
    $("corporateUsageLegend").innerHTML = "";
    $("corporateUsageChart").style.background = "var(--panel-2)";
    $("corporateUsageChart").innerHTML = "<span>0<small>credits</small></span>";
    return;
  }
  const models = data.models || [];
  const members = data.members || [];
  $("corporateCompanyName").textContent = data.company.name || "Company usage";
  $("corporateTotalCredits").textContent = `${Number(data.total_credits || 0).toLocaleString()}`;
  $("corporateMemberCount").textContent = String(members.length);
  $("corporateTopModality").textContent = topCompanyModality(models);
  $("corporateMemberRows").innerHTML =
    members
      .map((member) => `<tr>
        <td><strong>${escapeHTML(member.name)}</strong></td>
        <td>${escapeHTML(formatUserType(member.user_type))}</td>
        <td>${escapeHTML(member.email)}</td>
        <td>${Number(member.credits_used || 0).toLocaleString()}</td>
      </tr>`)
      .join("") || "<tr><td colspan=\"4\">No company members found.</td></tr>";
  renderCorporateUsageChart(models);
}

function topCompanyModality(models: CompanyUsageModel[]) {
  const totals = models.reduce<Record<string, number>>((acc, model) => {
    const modality = model.modality || "unknown";
    acc[modality] = (acc[modality] || 0) + Number(model.credits_used || 0);
    return acc;
  }, {});
  const [top] = Object.entries(totals).sort((a, b) => b[1] - a[1]);
  return top ? formatUserType(top[0]) : "-";
}

function renderCorporateUsageChart(models: CompanyUsageModel[]) {
  const colors = ["#3f8cff", "#12b981", "#f59e0b", "#ec4899", "#8b5cf6"];
  const total = models.reduce((sum, model) => sum + Number(model.credits_used || 0), 0) || 1;
  let cursor = 0;
  const gradient =
    models.length > 0
      ? models
          .map((model, index) => {
            const start = cursor;
            cursor += (Number(model.credits_used || 0) / total) * 100;
            return `${colors[index % colors.length]} ${start.toFixed(2)}% ${cursor.toFixed(2)}%`;
          })
          .join(", ")
      : "var(--panel-2) 0% 100%";
  $("corporateUsageChart").style.background = `conic-gradient(${gradient})`;
  $("corporateUsageChart").innerHTML = `<span>${Number(total === 1 && models.length === 0 ? 0 : total).toLocaleString()}<small>credits</small></span>`;
  $("corporateUsageLegend").innerHTML =
    models
      .map((model, index) => {
        const percent = Math.round((Number(model.credits_used || 0) / total) * 100);
        return `<div><i style="background:${colors[index % colors.length]}"></i><span>${escapeHTML(model.model)}</span><strong>${percent}%</strong></div>`;
      })
      .join("") || "<div>No model usage yet.</div>";
}

function formatUserType(value: string) {
  return value
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

async function loadPricing() {
  try {
    const data = await request<{ pricing: PricingRow[] }>("/api/v1/pricing");
    pricingRows = data.pricing;
  } catch {
    pricingRows = models.map((model) => ({
      provider: model.provider,
      model: model.name,
      model_slug: model.slug,
      modality: model.modality,
      profile: model.profile_name,
      profile_slug: model.profile_slug,
      input_price: 0,
      input_price_unit: "request",
      output_price: 0,
      output_price_unit: "request",
      currency: "CREDIT"
    }));
  }
  renderPricing();
}

function formatPrice(value: number, unit: string, currency: string) {
  const amount = Number(value) === 0 ? "0" : Number(value).toLocaleString(undefined, { maximumFractionDigits: 8 });
  return `${amount} ${currency} / ${unit}`;
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
  if (nextTab === "corporate-admin") {
    const user = getStoredUser();
    if (user) void loadCorporateUsage(user);
  }
  if (nextTab === "credit-usage") {
    renderUserCreditUsage();
  }
}

function openLogin() {
  setSignupMode(false);
  $("loginModal").classList.remove("hidden");
  closeAccountMenu();
  setAuthMessage("");
  $("authUsername").focus();
}

function closeLogin() {
  $("loginModal").classList.add("hidden");
}

function openSettings() {
  closeAccountMenu();
  ($("settingsLanguage") as HTMLSelectElement).value = localStorage.getItem("language") || "en";
  ($("settingsTheme") as HTMLSelectElement).value = localStorage.getItem("themeMode") || "system";
  ($("settingsUsageRange") as HTMLSelectElement).value = localStorage.getItem("defaultUsageRange") || (($("creditRange") as HTMLSelectElement).value || "30");
  ($("settingsWorkbenchType") as HTMLSelectElement).value = localStorage.getItem("defaultWorkbenchType") || workbenchModality || "image";
  ($("settingsEmailNotifications") as HTMLInputElement).checked = localStorage.getItem("emailNotifications") === "true";
  ($("settingsCompactTables") as HTMLInputElement).checked = localStorage.getItem("compactTables") === "true";
  $("settingsMessage").textContent = "";
  $("settingsModal").classList.remove("hidden");
}

function closeSettings() {
  $("settingsModal").classList.add("hidden");
}

function saveSettings(event: SubmitEvent) {
  event.preventDefault();
  const language = ($("settingsLanguage") as HTMLSelectElement).value;
  const theme = ($("settingsTheme") as HTMLSelectElement).value;
  const usageRange = (($("settingsUsageRange") as HTMLSelectElement).value || "30") as CreditRange;
  const defaultWorkbenchType = ($("settingsWorkbenchType") as HTMLSelectElement).value;
  applyLanguage(language);
  applyTheme(theme);
  localStorage.setItem("defaultUsageRange", usageRange);
  localStorage.setItem("defaultWorkbenchType", defaultWorkbenchType);
  localStorage.setItem("emailNotifications", String(($("settingsEmailNotifications") as HTMLInputElement).checked));
  applyCompactTables(($("settingsCompactTables") as HTMLInputElement).checked);
  ($("creditRange") as HTMLSelectElement).value = usageRange;
  ($("userUsageRange") as HTMLSelectElement).value = usageRange;
  workbenchModality = defaultWorkbenchType;
  document.querySelectorAll<HTMLButtonElement>("[data-workbench-modality]").forEach((button) => {
    button.classList.toggle("active", button.dataset.workbenchModality === workbenchModality);
  });
  renderCreditAnalytics();
  renderUserCreditUsage();
  renderModelControls();
  $("settingsMessage").textContent = "Settings saved.";
}

function openProfile() {
  closeAccountMenu();
  const user = getStoredUser();
  if (!user) {
    openLogin();
    return;
  }
  $("profileSummary").innerHTML = [
    profileRow("User name", user.name || "-"),
    profileRow("Email", user.email || "-"),
    profileRow("Account type", formatUserType(user.user_type || "individual_consumer")),
    profileRow("Company", user.company_name || "Individual account"),
    profileRow("User ID", user.id || "-"),
    profileRow("Current project", currentProjectID || "-")
  ].join("");
  $("profileModal").classList.remove("hidden");
}

function closeProfile() {
  $("profileModal").classList.add("hidden");
}

function profileRow(label: string, value: string) {
  return `<div class="profile-row"><span>${escapeHTML(label)}</span><strong>${escapeHTML(value)}</strong></div>`;
}

function setSignupMode(enabled: boolean) {
  signupMode = enabled;
  changePasswordMode = false;
  renderAuthMode();
}

function setChangePasswordMode(enabled: boolean) {
  changePasswordMode = enabled;
  signupMode = false;
  renderAuthMode();
}

function renderAuthMode() {
  $("nameField").classList.toggle("hidden", !signupMode);
  $("accountTypeField").classList.toggle("hidden", !signupMode);
  $("companyNameField").classList.toggle("hidden", !signupMode || selectedSignupAccountType() !== "corporate");
  $("confirmPasswordField").classList.toggle("hidden", !signupMode && !changePasswordMode);
  document.querySelector<HTMLElement>(".social-login")?.classList.toggle("hidden", changePasswordMode);
  ($("authUsername") as HTMLInputElement).autocomplete = signupMode || changePasswordMode ? "off" : "username";
  ($("authPassword") as HTMLInputElement).autocomplete = signupMode || changePasswordMode ? "new-password" : "current-password";
  ($("authConfirmPassword") as HTMLInputElement).autocomplete = "new-password";
  if (signupMode || changePasswordMode) {
    ($("authUsername") as HTMLInputElement).value = "";
    ($("authPassword") as HTMLInputElement).value = "";
    ($("authConfirmPassword") as HTMLInputElement).value = "";
  }
  if (changePasswordMode) {
    $("loginModeLabel").textContent = "Password";
    $("loginTitle").textContent = "Change password";
    $("authSubmit").textContent = "Update password";
    $("authSwitchText").textContent = "Back to login?";
    $("toggleSignup").textContent = "Login";
    $("changePasswordText").textContent = "Need a new account?";
    $("toggleChangePassword").textContent = "Sign up";
  } else {
    $("loginModeLabel").textContent = signupMode ? "Create account" : "Account";
    $("loginTitle").textContent = signupMode ? "Sign up for Model Market" : "Log in to Model Market";
    $("authSubmit").textContent = signupMode ? "Create account" : "Login";
    $("authSwitchText").textContent = signupMode ? "Already have an account?" : "No account yet?";
    $("toggleSignup").textContent = signupMode ? "Login" : "Sign up";
    $("changePasswordText").textContent = "Forgot or need to reset password?";
    $("toggleChangePassword").textContent = "Change password";
  }
  setAuthMessage("");
}

async function submitAuth(event: SubmitEvent) {
  event.preventDefault();
  const username = ($("authUsername") as HTMLInputElement).value.trim();
  const password = ($("authPassword") as HTMLInputElement).value;
  const confirmPassword = ($("authConfirmPassword") as HTMLInputElement).value;
  const name = ($("authName") as HTMLInputElement).value.trim();
  const accountType = selectedSignupAccountType();
  const companyName = ($("authCompanyName") as HTMLInputElement).value.trim();
  if ((signupMode || changePasswordMode) && password !== confirmPassword) {
    setAuthMessage("Passwords do not match.", true);
    return;
  }
  if (signupMode || changePasswordMode) {
    const passwordError = validateSignupPassword(password);
    if (passwordError) {
      setAuthMessage(passwordError, true);
      return;
    }
  }
  const path = changePasswordMode ? "/api/v1/auth/change-password" : signupMode ? "/api/v1/auth/signup" : "/api/v1/auth/login";
  const body = changePasswordMode ? { username, password } : signupMode ? { email: username, name, password, account_type: accountType, company_name: companyName } : { username, password };
  setAuthMessage(changePasswordMode ? "Updating password..." : signupMode ? "Creating account..." : "Logging in...");
  try {
    const auth = await request<AuthResponse>(path, {
      method: "POST",
      body: JSON.stringify(body)
    });
    if (changePasswordMode) {
      setAuthMessage("Password updated. You can log in with the new password.");
      window.setTimeout(() => setChangePasswordMode(false), 600);
      return;
    }
    completeAuth(auth, signupMode ? "Account created" : "Logged in");
  } catch (error) {
    setAuthMessage(formatAuthError(error), true);
  }
}

async function socialLogin(provider: string) {
  setAuthMessage(`Logging in with ${provider}...`);
  try {
    const auth = await request<AuthResponse>("/api/v1/auth/social/dev", {
      method: "POST",
      body: JSON.stringify({ provider })
    });
    completeAuth(auth, `Logged in with ${provider}`);
  } catch (error) {
    setAuthMessage(formatAuthError(error), true);
  }
}

function setAuthMessage(message: string, isError = false) {
  $("authMessage").textContent = message;
  $("authMessage").classList.toggle("error", isError);
}

function validateSignupPassword(password: string) {
  if (password.length < 8) return "Password must be at least 8 characters.";
  if (!/[A-Za-z]/.test(password)) return "Password must include at least one letter.";
  if (!/[0-9]/.test(password)) return "Password must include at least one number.";
  if (!/[^A-Za-z0-9]/.test(password)) return "Password must include at least one special character.";
  return "";
}

function formatAuthError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  if (message.includes("invalid_credentials")) return "Password is not correct.";
  if (message.includes("missing_credentials")) return "Enter your username and password.";
  if (message.includes("missing_signup_fields")) return "Enter your email and password.";
  if (message.includes("user_not_found")) return "User was not found.";
  if (message.includes("company_not_found")) return "Company name was not found.";
  if (message.includes("missing_company_name")) return "Enter your company name.";
  return message;
}

function completeAuth(auth: AuthResponse, message: string) {
  if (!auth.user) {
    setAuthMessage("Login response did not include a user.", true);
    return;
  }
  if (auth.project_id) currentProjectID = auth.project_id;
  localStorage.setItem("authUser", JSON.stringify(auth.user));
  localStorage.setItem("authSession", JSON.stringify(auth.session || {}));
  renderAuthUser();
  setAuthMessage(`${message} as ${auth.user.name}`);
  if (isAdminUser(auth.user)) {
    $("adminUserName").textContent = auth.user.name || auth.user.email;
    setActiveTab("admin");
  } else if (isCorporateAdmin(auth.user)) {
    void loadCorporateUsage(auth.user);
    setActiveTab("corporate-admin");
  }
  window.setTimeout(closeLogin, 550);
}

function renderAuthUser() {
  const user = getStoredUser();
  if (!user) {
    $("loginButton").textContent = "Login";
    $("loginButton").setAttribute("aria-expanded", "false");
    $("corporateAdminNav").classList.add("hidden");
    closeAccountMenu();
    return;
  }
  $("loginButton").textContent = user.name || user.email || "Account";
  $("corporateAdminNav").classList.toggle("hidden", !isCorporateAdmin(user));
  if (isAdminUser(user)) {
    $("adminUserName").textContent = user.name || user.email || "Admin User";
  }
  if (isCorporateAdmin(user)) {
    void loadCorporateUsage(user);
  }
}

function getStoredUser(): AuthUser | null {
  const raw = localStorage.getItem("authUser");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    localStorage.removeItem("authUser");
    localStorage.removeItem("authSession");
    return null;
  }
}

function selectedSignupAccountType() {
  return Array.from(document.querySelectorAll<HTMLInputElement>("input[name='accountType']")).find((input) => input.checked)?.value || "individual";
}

function updateSignupCompanyField() {
  const isCorporate = signupMode && selectedSignupAccountType() === "corporate";
  $("companyNameField").classList.toggle("hidden", !isCorporate);
}

function isAdminUser(user: { id?: string; email?: string; user_type?: string }) {
  return user.user_type === "sys_admin" || user.id === "user-admin" || user.email === "admin@example.com";
}

function isCorporateAdmin(user: { user_type?: string }) {
  return user.user_type === "corporate_admin";
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
  if (action === "settings") {
    openSettings();
    return;
  }
  if (action === "profile") {
    openProfile();
    return;
  }
  if (action === "logout") {
    localStorage.removeItem("authUser");
    localStorage.removeItem("authSession");
    renderAuthUser();
    setActiveTab("home");
    return;
  }
  setActiveTab(action);
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

function applyCompactTables(enabled: boolean) {
  document.documentElement.dataset.compactTables = String(enabled);
  localStorage.setItem("compactTables", String(enabled));
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
$("closeSettings").addEventListener("click", closeSettings);
$("settingsModal").addEventListener("click", (event) => {
  if (event.target === $("settingsModal")) closeSettings();
});
$("settingsForm").addEventListener("submit", (event) => saveSettings(event as SubmitEvent));
$("closeProfile").addEventListener("click", closeProfile);
$("profileModal").addEventListener("click", (event) => {
  if (event.target === $("profileModal")) closeProfile();
});
$("toggleSignup").addEventListener("click", () => {
  if (changePasswordMode) {
    setChangePasswordMode(false);
    return;
  }
  setSignupMode(!signupMode);
});
$("toggleChangePassword").addEventListener("click", () => {
  if (changePasswordMode) {
    setSignupMode(true);
    return;
  }
  setChangePasswordMode(true);
});
$("loginForm").addEventListener("submit", (event) => submitAuth(event as SubmitEvent));
document.querySelectorAll<HTMLInputElement>("input[name='accountType']").forEach((input) => {
  input.addEventListener("change", updateSignupCompanyField);
});
document.querySelectorAll<HTMLButtonElement>(".social-button").forEach((button) => {
  button.addEventListener("click", () => socialLogin(button.dataset.provider || ""));
});
$("createProject").addEventListener("click", () => createProject().catch(showError));
$("createConversation").addEventListener("click", () => createConversation().catch(showError));
$("projectFilter").addEventListener("focus", () => openProjectPicker(true));
$("projectFilter").addEventListener("input", () => openProjectPicker(false));
$("conversationFilter").addEventListener("focus", () => openConversationPicker(true));
$("conversationFilter").addEventListener("input", () => openConversationPicker(false));
$("projects").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-project-id]");
  if (!button) return;
  currentProjectID = button.dataset.projectId || currentProjectID;
  currentAPIKey = "";
  currentConversationID = "";
  currentBranchID = "";
  closeProjectPicker();
  renderProjects();
  loadWorkspace().catch(showError);
});
$("conversations").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-conversation-id]");
  if (!button) return;
  currentConversationID = button.dataset.conversationId || currentConversationID;
  currentBranchID = "";
  closeConversationPicker();
  loadConversationBranches()
    .then(loadConversationMessages)
    .catch(showError);
});
$("branchSelect").addEventListener("change", (event) => {
  currentBranchID = (event.target as HTMLSelectElement).value;
  loadConversationMessages().catch(showError);
});
$("chatTranscript").addEventListener("contextmenu", (event) => {
  const message = (event.target as HTMLElement).closest<HTMLElement>("[data-message-id]");
  if (!message) return;
  event.preventDefault();
  openMessageContextMenu(message.dataset.messageId || "", event.clientX, event.clientY);
});
$("branchFromMessage").addEventListener("click", () => startBranchFromMessage().catch(showError));
$("sendPrompt").addEventListener("click", () => sendPrompt().catch(showError));
$("uploadAsset").addEventListener("click", () => {
  ($("assetUploadInput") as HTMLInputElement).click();
});
$("assetUploadInput").addEventListener("change", (event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  uploadAssetFile(file).catch(showError);
  input.value = "";
});
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
$("pricingSearch").addEventListener("input", renderPricing);
$("creditRange").addEventListener("change", renderCreditAnalytics);
$("userUsageRange").addEventListener("change", renderUserCreditUsage);
$("pricingRows").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-pricing-model-slug]");
  if (button) showModelDetail(button.dataset.pricingModelSlug || "");
});
document.querySelectorAll<HTMLButtonElement>("[data-pricing-modality]").forEach((button) => {
  button.addEventListener("click", () => {
    selectedPricingModality = button.dataset.pricingModality || "all";
    document.querySelectorAll<HTMLButtonElement>("[data-pricing-modality]").forEach((item) => {
      item.classList.toggle("active", item.dataset.pricingModality === selectedPricingModality);
    });
    renderPricing();
  });
});
$("workbenchModelFilter").addEventListener("focus", () => openWorkbenchPicker(true));
$("workbenchModelFilter").addEventListener("input", () => openWorkbenchPicker(false));
document.querySelectorAll<HTMLButtonElement>("[data-workbench-modality]").forEach((button) => {
  button.addEventListener("click", () => {
    workbenchModality = button.dataset.workbenchModality || "image";
    const next = models.find((model) => model.modality === workbenchModality);
    if (next) selectedWorkbenchModel = next.profile_slug || next.slug;
    openWorkbenchPicker(true);
  });
});
$("workbenchModelList").addEventListener("click", (event) => {
  const button = (event.target as HTMLElement).closest<HTMLButtonElement>("[data-workbench-model]");
  if (!button) return;
  selectedWorkbenchModel = button.dataset.workbenchModel || selectedWorkbenchModel;
  closeWorkbenchPicker();
});
document.addEventListener("click", (event) => {
  const target = event.target as Node;
  if (!$("messageContextMenu").contains(target)) closeMessageContextMenu();
  const picker = document.querySelector(".workbench-model-picker");
  if (picker && !picker.contains(target)) closeWorkbenchPicker();
  const projectPicker = document.querySelector("#projectPicker");
  if (projectPicker && !projectPicker.contains(target)) closeProjectPicker();
  const conversationPicker = document.querySelector("#conversationPicker");
  if (conversationPicker && !conversationPicker.contains(target)) closeConversationPicker();
  const footerMenu = (event.target as HTMLElement).closest(".footer-menu");
  if (!footerMenu) closeFooterMenus();
});
document.querySelectorAll<HTMLButtonElement>("[data-footer-menu]").forEach((button) => {
  button.addEventListener("click", () => {
    const menu = button.closest(".footer-menu");
    const wasOpen = menu?.classList.contains("open") || false;
    closeFooterMenus();
    if (menu && !wasOpen) {
      menu.classList.add("open");
      button.setAttribute("aria-expanded", "true");
    }
  });
});
document.querySelectorAll<HTMLElement>(".footer-submenu a").forEach((link) => {
  link.addEventListener("click", closeFooterMenus);
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
applyCompactTables(localStorage.getItem("compactTables") === "true");
workbenchModality = localStorage.getItem("defaultWorkbenchType") || workbenchModality;
($("creditRange") as HTMLSelectElement).value = localStorage.getItem("defaultUsageRange") || (($("creditRange") as HTMLSelectElement).value || "30");
($("userUsageRange") as HTMLSelectElement).value = localStorage.getItem("defaultUsageRange") || (($("userUsageRange") as HTMLSelectElement).value || "30");
renderAuthUser();
renderCreditAnalytics();
setActiveTab(window.location.hash.slice(1) || "home");
loadAll().catch(showError);
