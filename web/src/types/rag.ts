export interface ChatReq {
  agent_id: number;
  query: string;
}

export interface IngestReq {
  agent_id: number;
  text: string;
  chunk_size?: number;
  overlap?: number;
}
