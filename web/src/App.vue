<script setup lang="ts">
</script>

<template>
  <div class="min-h-screen bg-slate-100 text-slate-900">
    <header class="border-b bg-white">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <h1 class="text-xl font-bold">FlexiRAG 控制台</h1>
        <nav class="flex gap-2 text-sm">
          <RouterLink class="rounded bg-slate-200 px-3 py-1" to="/agents">Agent 花名册</RouterLink>
          <RouterLink class="rounded bg-slate-200 px-3 py-1" to="/agents/create">创建 Agent</RouterLink>
          <RouterLink class="rounded bg-slate-200 px-3 py-1" to="/rag/ingest">知识入库</RouterLink>
          <RouterLink class="rounded bg-slate-200 px-3 py-1" to="/rag/chat">问答</RouterLink>
        </nav>
      </div>
    </header>

    <main class="mx-auto max-w-6xl p-4">
      <RouterView />
    </main>
  </div>
</template>
