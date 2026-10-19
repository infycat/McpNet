<template>
  <div style="width: 86px; cursor: pointer" @click="toggle">
    {{ busy ? "处理中..." : running ? "停止监听" : "启动监听" }}
  </div>
</template>

<script>
import {CallGoDo} from "./CallbackEventsOn.js";

export default {
  data() {
    return {running: false, busy: false}
  },
  methods: {
    setRunning(value) {
      this.running = value
    },
    async toggle() {
      if (this.busy) return
      this.busy = true
      try {
        const command = this.running ? "停止监听" : "启动监听"
        const ok = await CallGoDo(command, null)
        if (ok) {
          this.running = command === "启动监听"
          if (!this.running) window.vm.IEProxy?.setEnabled(false)
        }
      } finally {
        this.busy = false
      }
    }
  },
  mounted() {
    window.vm.ListenerState = this
  }
}
</script>
