<script setup lang="ts">
import { onMounted, reactive, watch } from "vue";
import { ingest } from "../../api/rag";
import { useAgentStore } from "../../stores/agent";

const agentStore = useAgentStore();

const form = reactive({
  agent_id: 0,
  text: "",
  chunk_size: 300,
  overlap: 40
});
const state = reactive({
  busy: false,
  message: "",
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
  state.message = "";
  try {
    state.message = await ingest({ ...form });
  } catch (err: any) {
    state.error = err?.message || "入库失败";
  } finally {
    state.busy = false;
  }
}
</script>

<template>
  <section class="rounded bg-white p-4 shadow">
    <h2 class="mb-4 text-lg font-semibold">知识入库</h2>
    <div class="space-y-3">
      <select v-model.number="form.agent_id" class="w-full rounded border p-2">
        <option :value="0" disabled>请选择 Agent 名称</option>
        <option v-for="agent in agentStore.agents" :key="agent.agent_id || agent.id" :value="Number(agent.agent_id || agent.id)">
          {{ agent.name }}
        </option>
      </select>
      <p v-if="agentStore.agents.length === 0" class="text-sm text-amber-600">当前没有可用 Agent，请先到“创建 Agent”页面新增。</p>
      <textarea v-model="form.text" class="h-48 w-full rounded border p-2" placeholder="输入知识文本" />
      <div class="grid grid-cols-2 gap-3">
        <div class="space-y-1">
          <input
            v-model.number="form.chunk_size"
            type="number"
            min="1"
            class="w-full rounded border p-2"
            placeholder="chunk_size"
          />
          <p class="text-xs text-slate-500">chunk_size：每个文本分片的长度，建议 200-500。</p>
        </div>
        <div class="space-y-1">
          <input
            v-model.number="form.overlap"
            type="number"
            min="0"
            class="w-full rounded border p-2"
            placeholder="overlap"
          />
          <p class="text-xs text-slate-500">overlap：相邻分片重叠长度，必须小于 chunk_size。</p>
        </div>
      </div>
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <p v-if="state.message" class="text-sm text-emerald-600">{{ state.message }}</p>
      <button :disabled="state.busy || agentStore.agents.length === 0" class="rounded bg-blue-700 px-4 py-2 text-white disabled:cursor-not-allowed disabled:bg-slate-400" @click="onSubmit">
        {{ state.busy ? "提交中..." : "开始入库" }}
      </button>
    </div>
  </section>
</template>
