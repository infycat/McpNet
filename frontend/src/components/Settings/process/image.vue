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
      currentParams: null,
    }
  },
  created() {
    this.currentParams = this.params;
    this.PID = "" + this.params.data["PID"];
    this.checkList = this.params.data["box"] === 1;
    this.icon = this.params.data.ico || "";
    this.processName = this.params.value || this.params.data["进程名"] || "";
  },
  mounted() {

  }, methods: {
    refresh(params) {
      // AgGrid reuses renderer instances while replacing row data. Keep the
      // event handler bound to the refreshed row, not the params from create.
      this.currentParams = params
      this.PID = "" + params.data["PID"]
      this.checkList = params.data["box"] === 1
      this.icon = params.data.ico || ""
      this.processName = params.value || params.data["进程名"] || ""
      return true
    },
    onCheckChange(value) {
      const row = this.currentParams?.data || this.params.data
      row["box"] = value ? 1 : 0;

      console.log("onCheckChange", value, this.PID);
      // A process-name rule applies to every matching PID. If this row is
      // checked because of that rule, toggle the name rule itself; otherwise
      // keep the existing per-PID behavior.
      if (row.selectedByName) {
        CallGoDo("进程驱动添加进程名", {Name: this.processName, isSet: value})
      } else {
        CallGoDo("进程驱动添加PID", {PID: this.PID, isSelected: value})
      }

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
