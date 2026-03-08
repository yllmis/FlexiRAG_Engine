<script setup lang="ts">
import { reactive } from "vue";
import { chat } from "../../api/rag";

const form = reactive({
  agent_id: 1,
  query: ""
});
const state = reactive({
  busy: false,
  answer: "",
  error: ""
});

async function onSubmit() {
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
      <input v-model.number="form.agent_id" type="number" class="w-full rounded border p-2" placeholder="agent_id" />
      <textarea v-model="form.query" class="h-28 w-full rounded border p-2" placeholder="输入问题" />
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <button :disabled="state.busy" class="rounded bg-violet-700 px-4 py-2 text-white" @click="onSubmit">
        {{ state.busy ? "思考中..." : "提问" }}
      </button>
      <div v-if="state.answer" class="rounded border bg-slate-50 p-3">
        <p class="text-sm font-semibold">回答：</p>
        <p class="mt-1 whitespace-pre-wrap text-sm">{{ state.answer }}</p>
      </div>
    </div>
  </section>
</template>
