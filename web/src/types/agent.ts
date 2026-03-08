export interface Agent {
  id?: number;
  agent_id?: number;
  name: string;
  system_prompt: string;
}

export interface CreateAgentReq {
  name: string;
  system_prompt: string;
}

export interface UpdateAgentReq {
  name?: string;
  system_prompt?: string;
}

export interface ListAgentsData {
  agents: Agent[];
}
