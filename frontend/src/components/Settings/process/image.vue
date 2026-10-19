<script>
//

import {CallGoDo} from "../../CallbackEventsOn.js";

export default {
  props: ['params'],
  data() {
    return {
      checkList: false,
      PID: "null",
      icon: "",
      processName: "",
    }
  },
  created() {
    this.PID = "" + this.params.data["PID"];
    this.checkList = this.params.data["box"] === 1;
    this.icon = this.params.data.ico || "";
    this.processName = this.params.value || this.params.data["进程名"] || "";
  },
  mounted() {

  }, methods: {
    onCheckChange(value) {
      this.params.data["box"] = value ? 1 : 0;

      console.log("onCheckChange", value, this.PID);
      CallGoDo("进程驱动添加PID", {PID: this.PID, isSelected: value})

    }
  }
};
</script>
<template>
  <div class="pid-cell">
    <el-checkbox v-if="params.colDef.field === 'PID'" v-model="checkList" @change="onCheckChange">{{ PID }}</el-checkbox>
    <template v-else>
      <img v-if="icon" :src="icon" alt="" class="process-icon" />
      <span class="process-name">{{ processName }}</span>
    </template>
  </div>
</template>

<style scoped>
.pid-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  height: 100%;
}

.process-icon {
  width: 16px;
  height: 16px;
  object-fit: contain;
  flex: 0 0 16px;
}

.process-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
