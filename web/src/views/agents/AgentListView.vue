<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { createAgent, deleteAgent, updateAgent } from "../../api/agents";
import { ingest } from "../../api/rag";
import { useAgentStore } from "../../stores/agent";
import type { Agent } from "../../types/agent";

const store = useAgentStore();
const route = useRoute();
const router = useRouter();
const fileInput = ref<HTMLInputElement | null>(null);
const selectedAgentIds = ref<number[]>([]);

const perfAgentIds = computed(() =>
  store.agents
    .filter((agent) => agent.name.toLowerCase().startsWith("perf-agent"))
    .map((agent) => resolveAgentId(agent))
    .filter((id) => id > 0)
);

const allSelected = computed(
  () => store.agents.length > 0 && selectedAgentIds.value.length === store.agents.length
);

const drawerState = reactive({
  open: false,
  mode: "create" as "create" | "edit",
  busy: false,
  error: "",
  success: "",
  targetId: 0,
  form: {
    name: "",
    system_prompt: ""
  }
});

const knowledgeState = reactive({
  agent_id: 0,
  text: "",
  chunk_size: 300,
  overlap: 40,
  dragActive: false,
  busy: false,
  error: "",
  success: "",
  uploadedFileName: ""
});

const selectedAgent = computed(() =>
  store.agents.find((agent) => resolveAgentId(agent) === knowledgeState.agent_id) || null
);

onMounted(async () => {
  await store.fetchAgents();
  cleanupSelectedIds();
  if (store.selectedAgentId > 0) {
    knowledgeState.agent_id = store.selectedAgentId;
  }
  syncDrawerFromRoute();
});

watch(
  () => route.fullPath,
  () => {
    syncDrawerFromRoute();
  }
);

watch(
  () => store.selectedAgentId,
  (agentId) => {
    if (!knowledgeState.agent_id && agentId > 0) {
      knowledgeState.agent_id = agentId;
    }
  }
);

watch(
  () => store.agents,
  () => {
    cleanupSelectedIds();
  },
  { deep: true }
);

function resolveAgentId(agent: Agent) {
  return Number(agent.agent_id || agent.id || 0);
}

function promptSummary(systemPrompt: string) {
  if (systemPrompt.length <= 72) {
    return systemPrompt;
  }
  return `${systemPrompt.slice(0, 72)}...`;
}

function syncKnowledgeTarget(agent: Agent) {
  const agentId = resolveAgentId(agent);
  store.selectAgent(agentId);
  knowledgeState.agent_id = agentId;
  document.getElementById("knowledge")?.scrollIntoView({ behavior: "smooth", block: "start" });
}

function isSelected(agent: Agent) {
  return selectedAgentIds.value.includes(resolveAgentId(agent));
}

function toggleAgentSelection(agent: Agent, checked: boolean) {
  const id = resolveAgentId(agent);
  if (id <= 0) {
    return;
  }
  if (checked) {
    if (!selectedAgentIds.value.includes(id)) {
      selectedAgentIds.value = [...selectedAgentIds.value, id];
    }
    return;
  }
  selectedAgentIds.value = selectedAgentIds.value.filter((item) => item !== id);
}

function toggleSelectAll(checked: boolean) {
  if (!checked) {
    selectedAgentIds.value = [];
    return;
  }
  selectedAgentIds.value = store.agents.map((agent) => resolveAgentId(agent)).filter((id) => id > 0);
}

function selectPerfAgents() {
  selectedAgentIds.value = [...perfAgentIds.value];
  if (selectedAgentIds.value.length === 0) {
    knowledgeState.error = "当前没有可批量删除的 perf-agent";
  }
}

function cleanupSelectedIds() {
  const validIds = new Set(store.agents.map((agent) => resolveAgentId(agent)).filter((id) => id > 0));
  selectedAgentIds.value = selectedAgentIds.value.filter((id) => validIds.has(id));
}

async function batchRemoveSelectedAgents() {
  if (selectedAgentIds.value.length === 0) {
    knowledgeState.error = "请先勾选要删除的 Agent";
    return;
  }
  const ok = window.confirm(`确认批量删除 ${selectedAgentIds.value.length} 个 Agent 吗？`);
  if (!ok) {
    return;
  }

  knowledgeState.error = "";
  knowledgeState.success = "";
  const targets = [...selectedAgentIds.value];
  let successCount = 0;
  let failedCount = 0;

  for (const id of targets) {
    try {
      await deleteAgent(id);
      successCount++;
      if (knowledgeState.agent_id === id) {
        knowledgeState.agent_id = 0;
      }
    } catch {
      failedCount++;
    }
  }

  await store.fetchAgents();
  cleanupSelectedIds();

  if (failedCount > 0) {
    knowledgeState.error = `批量删除完成：成功 ${successCount}，失败 ${failedCount}`;
    return;
  }
  knowledgeState.success = `批量删除完成：成功 ${successCount} 个 Agent`;
}

function openCreateDrawer() {
  drawerState.open = true;
  drawerState.mode = "create";
  drawerState.targetId = 0;
  drawerState.error = "";
  drawerState.success = "";
  drawerState.form.name = "";
  drawerState.form.system_prompt = "";
}

function openEditDrawer(agent: Agent) {
  drawerState.open = true;
  drawerState.mode = "edit";
  drawerState.targetId = resolveAgentId(agent);
  drawerState.error = "";
  drawerState.success = "";
  drawerState.form.name = agent.name;
  drawerState.form.system_prompt = agent.system_prompt;
}

function closeDrawer() {
  drawerState.open = false;
  drawerState.busy = false;
  drawerState.error = "";
  drawerState.success = "";
  if (route.query.drawer || route.query.id) {
    router.replace({ path: "/dashboard" });
  }
}

function syncDrawerFromRoute() {
  if (route.path !== "/dashboard") {
    return;
  }
  const mode = String(route.query.drawer || "");
  const targetId = Number(route.query.id || 0);
  if (mode === "create") {
    openCreateDrawer();
    return;
  }
  if (mode === "edit" && targetId > 0) {
    const target = store.agents.find((agent) => resolveAgentId(agent) === targetId);
    if (target) {
      openEditDrawer(target);
      return;
    }
  }
  if (drawerState.open && !route.query.drawer) {
    closeDrawer();
  }
}

async function submitDrawer() {
  drawerState.busy = true;
  drawerState.error = "";
  drawerState.success = "";
  try {
    if (drawerState.mode === "create") {
      await createAgent({
        name: drawerState.form.name,
        system_prompt: drawerState.form.system_prompt
      });
      drawerState.success = "Agent 创建成功";
    } else {
      await updateAgent(drawerState.targetId, {
        name: drawerState.form.name,
        system_prompt: drawerState.form.system_prompt
      });
      drawerState.success = "提示词与名称已更新";
    }
    await store.fetchAgents();
    if (store.selectedAgentId > 0 && !knowledgeState.agent_id) {
      knowledgeState.agent_id = store.selectedAgentId;
    }
    cleanupSelectedIds();
    window.setTimeout(() => {
      closeDrawer();
    }, 500);
  } catch (err: any) {
    drawerState.error = err?.message || (drawerState.mode === "create" ? "创建失败" : "更新失败");
  } finally {
    drawerState.busy = false;
  }
}

async function removeAgent(agent: Agent) {
  const id = resolveAgentId(agent);
  if (id <= 0) {
    return;
  }
  const ok = window.confirm(`确认删除 Agent #${id}（${agent.name}）吗？`);
  if (!ok) {
    return;
  }
  try {
    await deleteAgent(id);
    if (knowledgeState.agent_id === id) {
      knowledgeState.agent_id = 0;
    }
    await store.fetchAgents();
    cleanupSelectedIds();
    knowledgeState.success = `已删除 Agent #${id}`;
  } catch (err: any) {
    knowledgeState.error = err?.message || "删除 Agent 失败";
  }
}

function chooseFile() {
  fileInput.value?.click();
}

async function onFilePicked(event: Event) {
  const inputElement = event.target as HTMLInputElement;
  const file = inputElement.files?.[0];
  if (!file) {
    return;
  }
  await loadTextFile(file);
  inputElement.value = "";
}

async function onFileDrop(event: DragEvent) {
  knowledgeState.dragActive = false;
  const file = event.dataTransfer?.files?.[0];
  if (!file) {
    return;
  }
  await loadTextFile(file);
}

async function loadTextFile(file: File) {
  try {
    const content = await file.text();
    knowledgeState.text = content;
    knowledgeState.uploadedFileName = file.name;
    knowledgeState.error = "";
  } catch {
    knowledgeState.error = "文件读取失败，请改为直接粘贴文本。";
  }
}

async function submitKnowledge() {
  if (knowledgeState.agent_id <= 0) {
    knowledgeState.error = "请先选择要喂养的 Agent";
    return;
  }
  knowledgeState.busy = true;
  knowledgeState.error = "";
  knowledgeState.success = "";
  try {
    knowledgeState.success = await ingest({
      agent_id: knowledgeState.agent_id,
      text: knowledgeState.text,
      chunk_size: knowledgeState.chunk_size,
      overlap: knowledgeState.overlap
    });
    store.selectAgent(knowledgeState.agent_id);
  } catch (err: any) {
    knowledgeState.error = err?.message || "知识入库失败";
  } finally {
    knowledgeState.busy = false;
  }
}
</script>

<template>
  <section class="space-y-4">
    <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div class="rounded-[32px] border border-white/60 bg-white/85 p-5 shadow-[0_24px_80px_-38px_rgba(15,23,42,0.45)] backdrop-blur sm:p-6">
        <div class="flex flex-col gap-4 border-b border-slate-200 pb-5 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.26em] text-slate-400">Agent Dashboard</p>
            <h2 class="mt-2 text-2xl font-semibold text-slate-950">智能体卡片墙</h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-500">
              平铺展示当前所有 Agent，可快速创建、修改提示词，并与知识库面板联动。
            </p>
          </div>
          <div class="flex gap-2">
            <button class="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700" @click="store.fetchAgents()">刷新列表</button>
            <button class="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700" @click="selectPerfAgents()">选中 perf-agent</button>
            <button
              class="rounded-full border border-rose-200 bg-rose-50 px-4 py-2 text-sm font-medium text-rose-700 disabled:cursor-not-allowed disabled:border-slate-200 disabled:bg-slate-100 disabled:text-slate-400"
              :disabled="selectedAgentIds.length === 0"
              @click="batchRemoveSelectedAgents()"
            >
              批量删除（{{ selectedAgentIds.length }}）
            </button>
            <button class="rounded-full bg-slate-950 px-4 py-2 text-sm font-medium text-white" @click="openCreateDrawer()">创建 Agent</button>
          </div>
        </div>

        <div class="mt-4 flex items-center gap-3 text-sm text-slate-600">
          <input id="select-all-agents" :checked="allSelected" type="checkbox" class="h-4 w-4" @change="toggleSelectAll(($event.target as HTMLInputElement).checked)" />
          <label for="select-all-agents" class="cursor-pointer">全选当前列表</label>
          <span>已选 {{ selectedAgentIds.length }} / {{ store.agents.length }}</span>
        </div>

        <div v-if="store.loading" class="py-12 text-center text-sm text-slate-500">Agent 加载中...</div>

        <div v-else class="mt-6 grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
          <article
            v-for="agent in store.agents"
            :key="resolveAgentId(agent)"
            class="flex h-full flex-col rounded-[28px] border border-slate-200 bg-slate-50/80 p-5 transition hover:-translate-y-0.5 hover:shadow-lg"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <label class="inline-flex items-center gap-2 text-xs text-slate-500">
                  <input :checked="isSelected(agent)" type="checkbox" class="h-4 w-4" @change="toggleAgentSelection(agent, ($event.target as HTMLInputElement).checked)" />
                  选择
                </label>
                <h3 class="mt-2 text-xl font-semibold text-slate-950">{{ agent.name }}</h3>
              </div>
              <div class="flex items-center gap-2">
                <button class="rounded-full bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white" @click="openEditDrawer(agent)">
                  修改提示词
                </button>
                <button class="rounded-full bg-rose-600 px-3 py-1.5 text-sm font-medium text-white" @click="removeAgent(agent)">
                  删除
                </button>
              </div>
            </div>

            <p class="mt-4 flex-1 text-sm leading-7 text-slate-600">{{ promptSummary(agent.system_prompt) }}</p>

            <div class="mt-5 grid grid-cols-2 gap-3 text-sm text-slate-500">
              <span class="inline-flex min-h-10 w-full items-center justify-center rounded-full bg-white px-4 py-2 text-center font-medium text-slate-700">已接入问答台</span>
              <button
                class="inline-flex min-h-10 w-full items-center justify-center rounded-full border border-slate-200 bg-white px-4 py-2 text-center font-medium text-slate-900 transition hover:border-slate-300 hover:bg-slate-100"
                @click="syncKnowledgeTarget(agent)"
              >
                同步到知识面板
              </button>
            </div>
          </article>

          <article v-if="store.agents.length === 0" class="rounded-[28px] border border-dashed border-slate-300 bg-slate-50 p-8 text-center text-sm text-slate-500 md:col-span-2 2xl:col-span-3">
            当前没有可管理的 Agent，先创建一个用于问答和知识入库。
          </article>
        </div>
      </div>

      <aside id="knowledge" class="rounded-[32px] border border-slate-200/70 bg-slate-950 p-5 text-white shadow-[0_24px_80px_-40px_rgba(15,23,42,0.85)] sm:p-6">
        <p class="text-xs font-semibold uppercase tracking-[0.28em] text-slate-400">Knowledge Base</p>
        <h2 class="mt-2 text-2xl font-semibold">知识库面板</h2>
        <p class="mt-2 text-sm leading-6 text-slate-400">支持拖拽文本文件或直接粘贴长文，为指定 Agent 喂知识。</p>

        <div class="mt-5 space-y-4">
          <select v-model.number="knowledgeState.agent_id" class="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-white outline-none">
            <option :value="0" disabled class="text-slate-900">请选择 Agent</option>
            <option v-for="agent in store.agents" :key="resolveAgentId(agent)" :value="resolveAgentId(agent)" class="text-slate-900">
              {{ agent.name }}
            </option>
          </select>

          <div
            class="rounded-[28px] border border-dashed px-5 py-8 text-center transition"
            :class="knowledgeState.dragActive ? 'border-amber-300 bg-amber-300/10' : 'border-white/15 bg-white/5'"
            @dragenter.prevent="knowledgeState.dragActive = true"
            @dragover.prevent="knowledgeState.dragActive = true"
            @dragleave.prevent="knowledgeState.dragActive = false"
            @drop.prevent="onFileDrop"
          >
            <input ref="fileInput" type="file" class="hidden" accept=".txt,.md,.json,.csv" @change="onFilePicked" />
            <p class="text-lg font-semibold">拖拽长文本文件到这里</p>
            <p class="mt-2 text-sm text-slate-400">也可以直接点击下方按钮选择文件，读取后会自动填入文本框。</p>
            <button class="mt-4 rounded-full bg-white px-4 py-2 text-sm font-medium text-slate-950" @click="chooseFile">选择文件</button>
            <p v-if="knowledgeState.uploadedFileName" class="mt-3 text-xs text-emerald-300">已载入：{{ knowledgeState.uploadedFileName }}</p>
          </div>

          <div v-if="selectedAgent" class="rounded-2xl bg-white/5 px-4 py-3 text-sm text-slate-300">
            当前投喂对象：{{ selectedAgent.name }}
          </div>

          <textarea
            v-model="knowledgeState.text"
            class="h-56 w-full rounded-[28px] border border-white/10 bg-white/5 px-4 py-4 text-sm leading-7 text-white outline-none placeholder:text-slate-500"
            placeholder="粘贴知识文本，或通过上方拖拽区导入文本文件。"
          />

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="space-y-2 text-sm text-slate-300">
              <span>chunk_size</span>
              <input v-model.number="knowledgeState.chunk_size" type="number" min="1" class="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none" />
            </label>
            <label class="space-y-2 text-sm text-slate-300">
              <span>overlap</span>
              <input v-model.number="knowledgeState.overlap" type="number" min="0" class="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none" />
            </label>
          </div>

          <p class="text-xs leading-6 text-slate-400">推荐将 chunk_size 控制在 200-500，且 overlap 小于 chunk_size，以兼顾召回与冗余。</p>
          <p v-if="knowledgeState.error" class="text-sm text-rose-300">{{ knowledgeState.error }}</p>
          <p v-if="knowledgeState.success" class="text-sm text-emerald-300">{{ knowledgeState.success }}</p>

          <button
            :disabled="knowledgeState.busy || store.agents.length === 0"
            class="w-full rounded-full bg-amber-400 px-4 py-3 text-sm font-semibold text-slate-950 disabled:cursor-not-allowed disabled:bg-slate-500 disabled:text-slate-200"
            @click="submitKnowledge"
          >
            {{ knowledgeState.busy ? '入库中...' : '开始知识入库' }}
          </button>
        </div>
      </aside>
    </div>

    <div v-if="drawerState.open" class="fixed inset-0 z-50 flex justify-end bg-slate-950/35 backdrop-blur-sm" @click.self="closeDrawer()">
      <div class="flex h-full w-full max-w-xl flex-col bg-white p-6 shadow-2xl sm:p-8">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 pb-5">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.26em] text-slate-400">Drawer</p>
            <h2 class="mt-2 text-2xl font-semibold text-slate-950">
              {{ drawerState.mode === 'create' ? '创建 Agent' : '编辑 Agent' }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-slate-500">
              {{ drawerState.mode === 'create' ? '从右侧抽屉快速创建新智能体。' : '修改名称和系统提示词，立即影响问答体验。' }}
            </p>
          </div>
          <button class="rounded-full border border-slate-200 px-3 py-1.5 text-sm text-slate-600" @click="closeDrawer()">关闭</button>
        </div>

        <div class="flex-1 space-y-5 overflow-y-auto py-6">
          <label class="block space-y-2 text-sm text-slate-700">
            <span class="font-medium">Agent 名称</span>
            <input v-model="drawerState.form.name" class="w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none" placeholder="例如：教务小助手" />
          </label>

          <label class="block space-y-2 text-sm text-slate-700">
            <span class="font-medium">系统提示词</span>
            <textarea
              v-model="drawerState.form.system_prompt"
              class="h-72 w-full rounded-[28px] border border-slate-200 px-4 py-4 leading-7 outline-none"
              placeholder="输入角色设定、语气要求、边界规则与能力说明。"
            />
          </label>

          <div class="rounded-[28px] bg-slate-50 p-4 text-sm leading-7 text-slate-600">
            <p>建议把提示词拆成 3 段：角色定位、回答规则、禁止事项。这样便于后续维护和测试。</p>
          </div>
        </div>

        <div class="border-t border-slate-200 pt-5">
          <p v-if="drawerState.error" class="mb-3 text-sm text-rose-600">{{ drawerState.error }}</p>
          <p v-if="drawerState.success" class="mb-3 text-sm text-emerald-600">{{ drawerState.success }}</p>
          <div class="flex justify-end gap-3">
            <button class="rounded-full border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700" @click="closeDrawer()">取消</button>
            <button
              :disabled="drawerState.busy"
              class="rounded-full bg-slate-950 px-5 py-2.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-slate-300"
              @click="submitDrawer"
            >
              {{ drawerState.busy ? '提交中...' : drawerState.mode === 'create' ? '创建 Agent' : '保存修改' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
