import http from "./http";
import type { Agent, CreateAgentReq, ListAgentsData, UpdateAgentReq } from "../types/agent";
import type { ApiResponse } from "../types/common";

export async function createAgent(payload: CreateAgentReq): Promise<Agent> {
  const { data } = await http.post<ApiResponse<Agent>>("/api/v1/agents", payload);
  return data.data;
}

export async function listAgents(): Promise<Agent[]> {
  const { data } = await http.get<ApiResponse<ListAgentsData>>("/api/v1/agents");
  return data.data.agents;
}

export async function updateAgent(id: number, payload: UpdateAgentReq): Promise<Agent> {
  const { data } = await http.put<ApiResponse<Agent>>(`/api/v1/agents/${id}`, payload);
  return data.data;
}
