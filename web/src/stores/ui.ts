import { defineStore } from "pinia";

export const useUiStore = defineStore("ui", {
  state: () => ({
    busy: false,
    error: "",
    message: ""
  }),
  actions: {
    setError(msg: string) {
      this.error = msg;
      this.message = "";
    },
    setMessage(msg: string) {
      this.message = msg;
      this.error = "";
    },
    clearTips() {
      this.error = "";
      this.message = "";
    }
  }
});
