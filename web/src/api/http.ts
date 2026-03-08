import axios from "axios";
import type { ApiResponse } from "../types/common";

const http = axios.create({
  baseURL: "/",
  timeout: 15000
});

http.interceptors.response.use(
  (resp) => {
    const payload = resp.data as ApiResponse<unknown>;
    if (payload && typeof payload.code === "number" && payload.code !== 200) {
      return Promise.reject(new Error(payload.msg || "请求失败"));
    }
    return resp;
  },
  (err) => Promise.reject(err)
);

export default http;
