import axios from "axios";
import type { ApiResponse } from "../types/common";

const http = axios.create({
  baseURL: "/",
  timeout: 15000
});

let authTokenStateLogged = false;

function resolveAdminToken(): string {
  const raw = String(import.meta.env.VITE_ADMIN_TOKEN || "").trim();
  return raw.replace(/^['\"]|['\"]$/g, "");
}

function logAuthTokenState(token: string): void {
  if (authTokenStateLogged) {
    return;
  }
  authTokenStateLogged = true;

  if (!token) {
    console.warn("[auth] VITE_ADMIN_TOKEN 为空，写接口将返回 401。请检查 web/.env.local 并重启 npm run dev。");
    return;
  }

  console.info("[auth] VITE_ADMIN_TOKEN 已加载。鉴权请求头将自动注入。", { tokenLength: token.length });
}

http.interceptors.request.use((config) => {
  const token = resolveAdminToken();
  logAuthTokenState(token);
  if (token) {
    config.headers = config.headers || {};
    (config.headers as Record<string, string>).Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (resp) => {
    const payload = resp.data as ApiResponse<unknown>;
    if (payload && typeof payload.code === "number" && payload.code !== 200) {
      return Promise.reject(new Error(payload.msg || "请求失败"));
    }
    return resp;
  },
  (err) => {
    const status = err?.response?.status;
    if (status === 401) {
      return Promise.reject(new Error("未授权，请检查 VITE_ADMIN_TOKEN 配置"));
    }
    if (status === 429) {
      return Promise.reject(new Error("请求过于频繁，请稍后再试"));
    }
    return Promise.reject(err);
  }
);

export default http;
