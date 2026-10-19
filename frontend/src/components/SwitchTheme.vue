<template>
  <div style="height: 18px; width: 18px; cursor:pointer;">
    <i v-if="IsDark" @click="SwitchTheme" style="font-size: 20px;">
      <el-icon><Sunrise /></el-icon>
    </i>
    <i v-if="IsDark===false" @click="SwitchTheme" style="font-size: 20px;">
      <el-icon><Sunny /></el-icon>
    </i>
  </div>
</template>
<script>

import {CallGoDo} from "./CallbackEventsOn.js";

export default {
  components: {},
  computed: {
    IsDark() { return !this.Theme }
  },
  watch: {
    Theme(value) {
      if (window.ApplyAppTheme) window.ApplyAppTheme(value)
    }
  },
  data() {
    return {
      Theme: window.Theme.IsDark,
    }
  }, methods: {
    SwitchTheme() {
      this.Theme = !window.Theme.IsDark
      if (window.ApplyAppTheme) window.ApplyAppTheme(this.Theme)
      else window.Theme.IsDark = this.Theme
      CallGoDo("更新主题", {Dark: window.Theme.IsDark})
    }
  }, mounted() {
    window.vm.Theme = this
    if (window.ApplyAppTheme) window.ApplyAppTheme(window.Theme.IsDark)
  }
}
</script>
<style scoped>
</style>
