import http from "./http";
import type { ApiResponse } from "../types/common";
import type { ChatReq, IngestReq } from "../types/rag";

export async function chat(payload: ChatReq): Promise<string> {
  const { data } = await http.post<ApiResponse<{ answer: string }>>("/api/v1/chat", payload);
  return data.data.answer;
}

export async function ingest(payload: IngestReq): Promise<string> {
  const { data } = await http.post<ApiResponse<{ message: string }>>("/api/v1/knowledge/ingest", payload);
  return data.data.message;
}
