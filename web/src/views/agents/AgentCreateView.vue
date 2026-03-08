<script setup lang="ts">
import { reactive } from "vue";
import { useRouter } from "vue-router";
import { createAgent } from "../../api/agents";

const router = useRouter();
const form = reactive({
  name: "",
  system_prompt: ""
});
const state = reactive({
  busy: false,
  error: ""
});

async function onSubmit() {
  state.error = "";
  state.busy = true;
  try {
    await createAgent({ ...form });
    router.push("/agents");
  } catch (err: any) {
    state.error = err?.message || "创建失败";
  } finally {
    state.busy = false;
  }
}
</script>

<template>
  <section class="rounded bg-white p-4 shadow">
    <h2 class="mb-4 text-lg font-semibold">创建 Agent</h2>
    <div class="space-y-3">
      <input v-model="form.name" class="w-full rounded border p-2" placeholder="Agent 名称" />
      <textarea v-model="form.system_prompt" class="h-40 w-full rounded border p-2" placeholder="系统提示词" />
      <p v-if="state.error" class="text-sm text-red-600">{{ state.error }}</p>
      <button :disabled="state.busy" class="rounded bg-emerald-600 px-4 py-2 text-white" @click="onSubmit">
        {{ state.busy ? "提交中..." : "创建" }}
      </button>
    </div>
  </section>
</template>
