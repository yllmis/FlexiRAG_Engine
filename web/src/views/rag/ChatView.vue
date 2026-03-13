<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { chat } from "../../api/rag";
import { useAgentStore } from "../../stores/agent";
import type { Agent } from "../../types/agent";

type ChatMessage = {
  id: string;
  role: "assistant" | "user";
  content: string;
  timestamp: string;
};

const agentStore = useAgentStore();
const input = ref("");
const state = reactive({
  busy: false,
  error: ""
});
const conversations = reactive<Record<number, ChatMessage[]>>({});

const selectedAgentId = computed(() => agentStore.selectedAgentId);
const selectedAgent = computed(() =>
  agentStore.agents.find((agent) => resolveAgentId(agent) === selectedAgentId.value) || null
);
const activeMessages = computed(() => {
  const agentId = selectedAgentId.value;
  if (!agentId) {
    return [] as ChatMessage[];
  }
  ensureConversation(agentId);
  return conversations[agentId];
});

onMounted(async () => {
  await agentStore.fetchAgents();
  if (agentStore.selectedAgentId > 0) {
    ensureConversation(agentStore.selectedAgentId);
  }
});

watch(
  () => agentStore.selectedAgentId,
  (agentId) => {
    if (agentId > 0) {
      ensureConversation(agentId);
    }
  }
);

function resolveAgentId(agent: Agent) {
  return Number(agent.agent_id || agent.id || 0);
}

function initials(name: string) {
  return (name || "A").slice(0, 1).toUpperCase();
}

function createMessage(role: "assistant" | "user", content: string): ChatMessage {
  return {
    id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    role,
    content,
    timestamp: new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })
  };
}

function ensureConversation(agentId: number) {
  if (!conversations[agentId]) {
    const agent = agentStore.agents.find((item) => resolveAgentId(item) === agentId);
    conversations[agentId] = [
      createMessage(
        "assistant",
        agent ? `我是 ${agent.name}，可以直接开始提问。` : "请选择一个 Agent 开始对话。"
      )
    ];
  }
}

async function onSubmit() {
  const agentId = selectedAgentId.value;
  const query = input.value.trim();
  if (agentId <= 0) {
    state.error = "请先选择已创建的 Agent";
    return;
  }
  if (!query) {
    state.error = "请输入问题内容";
    return;
  }

  ensureConversation(agentId);
  conversations[agentId].push(createMessage("user", query));
  input.value = "";
  state.busy = true;
  state.error = "";
  try {
    const answer = await chat({ agent_id: agentId, query });
    conversations[agentId].push(createMessage("assistant", answer));
  } catch (err: any) {
    const message = err?.message || "问答失败";
    conversations[agentId].push(createMessage("assistant", `请求失败：${message}`));
    state.error = message;
  } finally {
    state.busy = false;
  }
}
</script>

<template>
  <section class="grid min-h-[calc(100vh-11rem)] gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
    <aside class="flex flex-col overflow-hidden rounded-[32px] border border-slate-200/80 bg-slate-950 text-white shadow-[0_24px_80px_-40px_rgba(15,23,42,0.85)]">
      <div class="border-b border-white/10 px-5 py-5">
        <p class="text-xs font-semibold uppercase tracking-[0.28em] text-slate-400">Sidebar</p>
        <h2 class="mt-2 text-2xl font-semibold">Agent 花名册</h2>
      </div>

      <div class="flex-1 space-y-2 overflow-y-auto px-3 py-4">
        <button
          v-for="agent in agentStore.agents"
          :key="resolveAgentId(agent)"
          class="flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-left transition"
          :class="resolveAgentId(agent) === selectedAgentId ? 'bg-white text-slate-950 shadow-lg' : 'bg-white/5 text-white hover:bg-white/10'"
          @click="agentStore.selectAgent(resolveAgentId(agent))"
        >
          <div
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl text-sm font-semibold"
            :class="resolveAgentId(agent) === selectedAgentId ? 'bg-amber-300 text-slate-950' : 'bg-white/10 text-white'"
          >
            {{ initials(agent.name) }}
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate pt-2 text-sm font-semibold">{{ agent.name }}</p>
          </div>
        </button>

        <div v-if="!agentStore.loading && agentStore.agents.length === 0" class="rounded-3xl border border-dashed border-white/15 px-4 py-8 text-center text-sm text-slate-400">
          当前还没有 Agent，请先去管理后台创建。
        </div>
      </div>
    </aside>

    <div class="flex min-h-0 flex-col overflow-hidden rounded-[32px] border border-white/60 bg-white/85 shadow-[0_24px_80px_-38px_rgba(15,23,42,0.45)] backdrop-blur">
      <header class="border-b border-slate-200 px-5 py-4 sm:px-6">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.26em] text-slate-400">Chat Console</p>
            <h2 class="mt-2 text-2xl font-semibold text-slate-950">
              {{ selectedAgent ? selectedAgent.name : "请选择 Agent" }}
            </h2>
          </div>
          <div class="rounded-2xl bg-slate-100 px-4 py-3 text-sm text-slate-600">
            <p>对话条数：{{ activeMessages.length }}</p>
            <p class="mt-1">当前上下文仅保存在前端视图内，便于切换 Agent 时快速恢复。</p>
          </div>
        </div>
      </header>

      <div class="flex-1 space-y-4 overflow-y-auto bg-[linear-gradient(180deg,rgba(248,250,252,0.9)_0%,rgba(241,245,249,0.9)_100%)] px-5 py-5 sm:px-6">
        <div v-for="message in activeMessages" :key="message.id" class="flex" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
          <div class="max-w-[85%] rounded-[24px] px-4 py-3 shadow-sm sm:max-w-[72%]" :class="message.role === 'user' ? 'bg-slate-950 text-white' : 'bg-white text-slate-800'">
            <p class="whitespace-pre-wrap text-sm leading-7">{{ message.content }}</p>
            <p class="mt-2 text-[11px]" :class="message.role === 'user' ? 'text-slate-300' : 'text-slate-400'">{{ message.timestamp }}</p>
          </div>
        </div>

        <div v-if="state.busy" class="flex justify-start">
          <div class="rounded-[24px] bg-white px-4 py-3 text-sm text-slate-500 shadow-sm">正在生成回答，请稍候...</div>
        </div>
      </div>

      <footer class="border-t border-slate-200 bg-white/90 px-4 py-4 sm:px-6">
        <p v-if="state.error" class="mb-3 text-sm text-rose-600">{{ state.error }}</p>
        <div class="rounded-[28px] border border-slate-200 bg-slate-50 p-3 shadow-inner shadow-slate-200/40">
          <div class="flex items-end gap-3">
            <button
              type="button"
              class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-slate-200 bg-white text-xl text-slate-500"
              disabled
              title="后续可扩展上传文件"
            >
              +
            </button>
            <textarea
              v-model="input"
              class="h-24 min-h-[6rem] flex-1 resize-none border-0 bg-transparent px-1 py-2 text-sm leading-7 text-slate-800 outline-none placeholder:text-slate-400"
              placeholder="输入你的问题，系统会按当前 Agent 的设定回复。"
              @keydown.enter.exact.prevent="onSubmit"
            />
            <button
              :disabled="state.busy || agentStore.agents.length === 0"
              class="rounded-2xl bg-amber-400 px-5 py-3 text-sm font-semibold text-slate-950 transition hover:bg-amber-300 disabled:cursor-not-allowed disabled:bg-slate-300"
              @click="onSubmit"
            >
              {{ state.busy ? "发送中" : "发送" }}
            </button>
          </div>
        </div>
      </footer>
    </div>
  </section>
</template>
