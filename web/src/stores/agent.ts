import { defineStore } from "pinia";
import { listAgents } from "../api/agents";
import type { Agent } from "../types/agent";

export const useAgentStore = defineStore("agent", {
  state: () => ({
    agents: [] as Agent[],
    selectedAgentId: 0,
    loading: false
  }),
  actions: {
    async fetchAgents() {
      this.loading = true;
      try {
        this.agents = await listAgents();
        const hasSelectedAgent = this.agents.some(
          (agent) => Number(agent.agent_id || agent.id || 0) === this.selectedAgentId
        );
        if ((!this.selectedAgentId || !hasSelectedAgent) && this.agents.length > 0) {
          this.selectedAgentId = Number(this.agents[0].agent_id || this.agents[0].id || 0);
        }
        if (this.agents.length === 0) {
          this.selectedAgentId = 0;
        }
      } finally {
        this.loading = false;
      }
    },
    selectAgent(agentId: number) {
      this.selectedAgentId = agentId;
    }
  }
});
