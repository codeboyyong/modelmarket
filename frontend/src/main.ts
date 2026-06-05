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

const apiBase = (window as Window & { API_BASE_URL?: string }).API_BASE_URL || "http://localhost:8080";
let currentProjectID = "";
let currentAPIKey = "";
let models: Model[] = [];

const $ = <T extends HTMLElement>(id: string): T => {
  const el = document.getElementById(id);
  if (!el) throw new Error(`missing element ${id}`);
  return el as T;
};

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
  const summary = await request<Summary>("/api/v1/dev/summary");
  $("summary").innerHTML = Object.entries(summary.counts)
    .map(([key, value]) => `<div><strong>${key}</strong><span>${value}</span></div>`)
    .join("");
}

async function loadProjects() {
  const data = await request<{ projects: Project[] }>("/api/v1/projects");
  const [project] = data.projects;
  if (project) currentProjectID = project.id;
  $("projects").innerHTML = data.projects
    .map(
      (project) => `<div>
        <strong>${project.name}</strong>
        <small>${project.organization}</small>
        <p>${project.paid_credits + project.promotional_credits} credits available</p>
      </div>`
    )
    .join("");
  if (currentProjectID) await loadAPIKeys();
}

async function loadModels() {
  const data = await request<{ models: Model[] }>("/api/v1/models");
  models = data.models;
  $("models").innerHTML = models
    .map(
      (model) => `<div class="card">
        <strong>${model.name}</strong>
        <small>${model.provider} / ${model.modality}</small>
        <p>Profile: ${model.profile_name || "None"}</p>
      </div>`
    )
    .join("");
  $("modelSelect").innerHTML = models
    .filter((model) => model.modality === "chat")
    .map((model) => `<option value="${model.profile_slug || model.slug}">${model.name}</option>`)
    .join("");
}

async function loadAPIKeys() {
  const data = await request<{ api_keys: Array<{ id: string; name: string; prefix: string; status: string }> }>(
    `/api/v1/api-keys?project_id=${encodeURIComponent(currentProjectID)}`
  );
  $("apiKeys").innerHTML =
    data.api_keys
      .map((key) => `<div><strong>${key.name}</strong><small>${key.prefix}... ${key.status}</small></div>`)
      .join("") || "<p class=\"muted\">No API keys yet.</p>";
}

async function createKey() {
  const data = await request<{ api_key: string; prefix: string }>("/api/v1/api-keys", {
    method: "POST",
    body: JSON.stringify({ project_id: currentProjectID, name: "Frontend dev key" })
  });
  currentAPIKey = data.api_key;
  $("newKey").innerHTML = `<p><strong>New key:</strong> <code>${data.api_key}</code></p>`;
  await loadAPIKeys();
}

async function sendPrompt() {
  if (!currentAPIKey) await createKey();
  const model = ($("modelSelect") as HTMLSelectElement).value || "mock-chat";
  const prompt = ($("prompt") as HTMLTextAreaElement).value;
  const data = await request("/api/v1/chat/completions", {
    method: "POST",
    headers: { Authorization: `Bearer ${currentAPIKey}` },
    body: JSON.stringify({ model, messages: [{ role: "user", content: prompt }] })
  });
  $("chatResult").textContent = JSON.stringify(data, null, 2);
  await loadSummary();
}

$("refresh").addEventListener("click", () => loadAll().catch(showError));
$("createKey").addEventListener("click", () => createKey().catch(showError));
$("sendPrompt").addEventListener("click", () => sendPrompt().catch(showError));

function showError(error: unknown) {
  $("chatResult").textContent = error instanceof Error ? error.message : String(error);
}

loadAll().catch(showError);
