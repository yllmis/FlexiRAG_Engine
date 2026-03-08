<script setup lang="ts">
import { reactive } from "vue";
import { ingest } from "../../api/rag";

const form = reactive({
  agent_id: 1,
  text: "",
  chunk_size: 300,
  overlap: 40
});
const state = reactive({
  busy: false,
  message: "",
  error: ""
});

async function onSubmit() {
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
      <input v-model.number="form.agent_id" type="number" class="w-full rounded border p-2" placeholder="agent_id" />
      <textarea v-model="form.text" class="h-48 w-full rounded border p-2" placeholder="输入知识文本" />
      <div class="grid grid-cols-2 gap-2">
        <input v-model.number="form.chunk_size" type="number" class="rounded border p-2" placeholder="chunk_size" />
        <input v-model.number="form.overlap" type="number" class="rounded border p-2" placeholder="overlap" />
      </div>
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <p v-if="state.message" class="text-sm text-emerald-600">{{ state.message }}</p>
      <button :disabled="state.busy" class="rounded bg-blue-700 px-4 py-2 text-white" @click="onSubmit">
        {{ state.busy ? "提交中..." : "开始入库" }}
      </button>
    </div>
  </section>
</template>
