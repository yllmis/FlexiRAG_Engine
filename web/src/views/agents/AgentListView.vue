<script setup lang="ts">
import { onMounted } from "vue";
import { useAgentStore } from "../../stores/agent";

const store = useAgentStore();

onMounted(async () => {
  await store.fetchAgents();
});
</script>

<template>
  <section class="rounded bg-white p-4 shadow">
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold">Agent 花名册</h2>
      <button class="rounded bg-slate-900 px-3 py-1 text-white" @click="store.fetchAgents()">刷新</button>
    </div>

    <p v-if="store.loading" class="text-sm text-slate-500">加载中...</p>
    <table v-else class="w-full table-auto border-collapse text-sm">
      <thead>
        <tr class="border-b text-left">
          <th class="py-2">ID</th>
          <th class="py-2">名称</th>
          <th class="py-2">系统提示词</th>
          <th class="py-2">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="agent in store.agents" :key="agent.agent_id || agent.id" class="border-b align-top">
          <td class="py-2">{{ agent.agent_id || agent.id }}</td>
          <td class="py-2">{{ agent.name }}</td>
          <td class="py-2">{{ agent.system_prompt }}</td>
          <td class="py-2">
            <RouterLink class="inline-block whitespace-nowrap rounded bg-indigo-600 px-3 py-1 text-white" :to="`/agents/${agent.agent_id || agent.id}/edit`">
              编辑
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
