<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const navItems = [
  { label: "问答台", to: "/chat", match: "/chat" },
  { label: "管理后台", to: "/dashboard", match: "/dashboard" }
];

const pageTitle = computed(() => (route.path.startsWith("/dashboard") ? "智能体管理后台" : "沉浸式问答台"));

function navClass(match: string) {
  return route.path.startsWith(match)
    ? "bg-slate-950 text-white shadow-lg shadow-slate-950/20"
    : "bg-white/70 text-slate-700 hover:bg-white hover:text-slate-950";
}
</script>

<template>
  <div class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(251,191,36,0.22),_transparent_28%),linear-gradient(180deg,_#f8fafc_0%,_#e2e8f0_100%)] text-slate-900">
    <div class="mx-auto flex min-h-screen max-w-7xl flex-col px-4 pb-6 pt-5 sm:px-6 lg:px-8">
      <header class="mb-6 rounded-[28px] border border-white/60 bg-white/70 px-5 py-4 shadow-[0_24px_80px_-36px_rgba(15,23,42,0.45)] backdrop-blur xl:px-7">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">FlexiRAG Engine</p>
            <div class="mt-2 flex flex-col gap-2 lg:flex-row lg:items-end lg:gap-4">
              <h1 class="text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">{{ pageTitle }}</h1>
              <p class="max-w-2xl text-sm leading-6 text-slate-600">
                C 端聚焦高频对话体验，B 端承接 Agent 配置、提示词维护与知识喂养。
              </p>
            </div>
          </div>

          <nav class="flex flex-wrap gap-2 text-sm font-medium">
            <RouterLink
              v-for="item in navItems"
              :key="item.to"
              :to="item.to"
              class="rounded-full px-4 py-2 transition"
              :class="navClass(item.match)"
            >
              {{ item.label }}
            </RouterLink>
          </nav>
        </div>
      </header>

      <main class="flex-1">
        <RouterView />
      </main>
    </div>
  </div>
</template>
