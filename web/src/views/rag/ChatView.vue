<script setup lang="ts">
import { onMounted, reactive, watch } from "vue";
import { chat } from "../../api/rag";
import { useAgentStore } from "../../stores/agent";

const agentStore = useAgentStore();

const form = reactive({
  agent_id: 0,
  query: ""
});
const state = reactive({
  busy: false,
  answer: "",
  error: ""
});

onMounted(async () => {
  await agentStore.fetchAgents();
  if (agentStore.selectedAgentId > 0) {
    form.agent_id = agentStore.selectedAgentId;
  }
});

watch(
  () => form.agent_id,
  (val) => {
    agentStore.selectedAgentId = val;
  }
);

async function onSubmit() {
  if (form.agent_id <= 0) {
    state.error = "请先选择已创建的 Agent";
    return;
  }

  state.busy = true;
  state.error = "";
  state.answer = "";
  try {
    state.answer = await chat({ ...form });
  } catch (err: any) {
    state.error = err?.message || "问答失败";
  } finally {
    state.busy = false;
  }
}
</script>

<template>
  <section class="rounded bg-white p-4 shadow">
    <h2 class="mb-4 text-lg font-semibold">问答</h2>
    <div class="space-y-3">
      <select v-model.number="form.agent_id" class="w-full rounded border p-2">
        <option :value="0" disabled>请选择 Agent 名称</option>
        <option v-for="agent in agentStore.agents" :key="agent.agent_id || agent.id" :value="Number(agent.agent_id || agent.id)">
          {{ agent.name }}
        </option>
      </select>
      <p v-if="agentStore.agents.length === 0" class="text-sm text-amber-600">当前没有可用 Agent，请先到“创建 Agent”页面新增。</p>
      <textarea v-model="form.query" class="h-28 w-full rounded border p-2" placeholder="输入问题" />
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <button :disabled="state.busy || agentStore.agents.length === 0" class="rounded bg-violet-700 px-4 py-2 text-white disabled:cursor-not-allowed disabled:bg-slate-400" @click="onSubmit">
        {{ state.busy ? "思考中..." : "提问" }}
      </button>
      <div v-if="state.answer" class="rounded border bg-slate-50 p-3">
        <p class="text-sm font-semibold">回答：</p>
        <p class="mt-1 whitespace-pre-wrap text-sm">{{ state.answer }}</p>
      </div>
    </div>
  </section>
</template>
