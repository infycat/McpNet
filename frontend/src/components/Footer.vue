<template>
  <div class="app-status-bar" :style="backStyle">
    <ListenerState/>
    <PartitionOperator/>
    <SetIEProxy/>
    <PartitionOperator/>
    <HookState/>
    <PartitionOperator/>
    <HookState2/>
    <PartitionOperator/>
    <span ref="eLabel">{{ Title }}</span>
    <SwitchTheme class="app-status-theme"/>
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
      return { backgroundColor: 'var(--app-header, #202020)', color: 'var(--app-text, #f5f5f5)', borderTop: '1px solid var(--app-border, #424242)' }
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
  }
}
</script>
<style scoped>
.app-status-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  width: auto;
  z-index: 2147483000;
  box-sizing: border-box;
  height: 30px;
  padding: 0 10px 0 16px;
  display: flex;
  align-items: center;
  gap: 0;
  overflow: hidden;
  white-space: nowrap;
  font-size: 13px;
  line-height: 1;
  background-color: #202020;
}

.app-status-bar :deep(img) {
  height: 14px !important;
  width: auto;
}

.app-status-bar :deep(svg) {
  transform: scale(.8);
  transform-origin: center;
}

.app-status-bar :deep(.app-status-theme) {
  flex: 0 0 18px;
  width: 18px;
  height: 18px;
  margin-left: auto;
}
</style>
