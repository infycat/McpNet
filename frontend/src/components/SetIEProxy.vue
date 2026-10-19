<template>
  &nbsp;&nbsp;&nbsp;
  <div v-show="isSettings===false" style="width: 100px;cursor: pointer; " @click="Click">{{ title }}</div>
  <div v-show="isSettings" style="width: 100px;cursor: pointer; ">正在设置中...</div>
</template>

<script>

import {CallGoDo} from "./CallbackEventsOn.js";

export default {
  data() {
    return {
      state: false,
      title: "未设置系统IE代理",
      isSettings: false
    }
  }, methods: {
    Click() {
      if (this.isSettings) {
        return
      }
      const newState = !this.state
      this.isSettings = true
      CallGoDo("设置IE代理", {Set: newState}).then(res => {
        this.isSettings = false
        if (!res) {
          return
        }
        this.setEnabled(newState)
        if (newState) window.vm.ListenerState?.setRunning(true)
      })
    },
    setEnabled(enabled) {
      this.state = enabled
      this.title = enabled ? "已设置系统IE代理" : "未设置系统IE代理"
    }
  }, mounted() {
    window.vm.IEProxy = this
  }
}
</script>
<style scoped>

</style>
