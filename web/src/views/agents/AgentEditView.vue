<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { useRoute } from "vue-router";
import { listAgents, updateAgent } from "../../api/agents";

const route = useRoute();
const state = reactive({
  busy: false,
  message: "",
  error: "",
  id: Number(route.params.id),
  name: "",
  system_prompt: ""
});

onMounted(async () => {
  try {
    const agents = await listAgents();
    const target = agents.find((a) => Number(a.agent_id || a.id) === state.id);
    if (target) {
      state.name = target.name;
      state.system_prompt = target.system_prompt;
    }
  } catch (err: any) {
    state.error = err?.message || "加载失败";
  }
});

async function onSubmit() {
  state.busy = true;
  state.error = "";
  state.message = "";
  try {
    await updateAgent(state.id, {
      name: state.name,
      system_prompt: state.system_prompt
    });
    state.message = "更新成功";
  } catch (err: any) {
    state.error = err?.message || "更新失败";
  } finally {
    state.busy = false;
  }
}
</script>

<template>
  <section class="rounded bg-white p-4 shadow">
    <h2 class="mb-4 text-lg font-semibold">编辑 Agent #{{ state.id }}</h2>
    <div class="space-y-3">
      <input v-model="state.name" class="w-full rounded border p-2" placeholder="Agent 名称" />
      <textarea v-model="state.system_prompt" class="h-40 w-full rounded border p-2" placeholder="系统提示词" />
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <p v-if="state.message" class="text-sm text-emerald-600">{{ state.message }}</p>
      <button :disabled="state.busy" class="rounded bg-indigo-600 px-4 py-2 text-white" @click="onSubmit">
        {{ state.busy ? "提交中..." : "保存" }}
      </button>
    </div>
  </section>
</template>
