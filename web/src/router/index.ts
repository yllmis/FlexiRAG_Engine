import { createRouter, createWebHistory } from "vue-router";

import AgentListView from "../views/agents/AgentListView.vue";
import ChatView from "../views/rag/ChatView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/chat" },
    { path: "/chat", component: ChatView },
    { path: "/dashboard", component: AgentListView },
    { path: "/agents", redirect: "/dashboard" },
    { path: "/agents/create", redirect: "/dashboard?drawer=create" },
    {
      path: "/agents/:id/edit",
      redirect: (to) => ({
        path: "/dashboard",
        query: { drawer: "edit", id: String(to.params.id || "") }
      })
    },
    { path: "/rag/ingest", redirect: "/dashboard#knowledge" },
    { path: "/rag/chat", redirect: "/chat" }
  ]
});

export default router;
