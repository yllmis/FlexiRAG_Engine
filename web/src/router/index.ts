import { createRouter, createWebHistory } from "vue-router";

import AgentListView from "../views/agents/AgentListView.vue";
import AgentCreateView from "../views/agents/AgentCreateView.vue";
import AgentEditView from "../views/agents/AgentEditView.vue";
import IngestView from "../views/rag/IngestView.vue";
import ChatView from "../views/rag/ChatView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/agents" },
    { path: "/agents", component: AgentListView },
    { path: "/agents/create", component: AgentCreateView },
    { path: "/agents/:id/edit", component: AgentEditView },
    { path: "/rag/ingest", component: IngestView },
    { path: "/rag/chat", component: ChatView }
  ]
});

export default router;
