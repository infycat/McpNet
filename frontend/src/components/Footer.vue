<template>
  <div class="ag-header-group-cell-label" :style="backStyle">
    <ListenerState/>
    <PartitionOperator/>
    <SetIEProxy/>
    <PartitionOperator/>
    <HookState/>
    <PartitionOperator/>
    <HookState2/>
    <PartitionOperator/>
    <span ref="eLabel">{{ Title }}</span>
    <div class="ag-header-cell-resize" style="position:absolute;height: 20px;width: 20px"></div>

    <SwitchTheme style="right: 6px;position:absolute;cursor:pointer;top:2px"/>
  </div>
</template>


<script>
import agOption from './option.vue';
import SwitchTheme from './SwitchTheme.vue';
import PartitionOperator from './PartitionOperator.vue';
import SetIEProxy from "./SetIEProxy.vue";
import ListenerState from "./ListenerState.vue";
import HookState from "./HookState.vue";
import HookState2 from "./HookState2.vue";

export default {
  components: {
    SetIEProxy,
    ListenerState,
    agOption, SwitchTheme, PartitionOperator, HookState, HookState2
  },
  computed: {
    backStyle() {
      let c = "height: 30px;background-color: #202020;position: fixed;left: 0;right: 0;bottom: 0;z-index: 999999;"
      if (!this.theme) {
        c += "background-color: #f0f0f0;"
      } else {
        c += ""
      }
      return c
    }
  },
  data() {
    return {
      Title: "正在启动...",
      get theme() {
        return window.Theme.IsDark
      },
      set theme(newValue) {
        window.Theme.IsDark = newValue
      }
    }
  }, methods: {},
  mounted() {
    window.vm["Footer"] = this
    document.body.appendChild(this.$el)
  }
}
</script>
<style scoped>
</style>
