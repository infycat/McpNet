<template>
  <div style="position: relative;text-align: left;left: 20px">
    <el-tag class="ml-2" type="success">MIT License</el-tag>
    <br><br>
  </div>
  <div style="position: relative;text-align: left;left: 20px">
    <el-text class="mx-1" type="primary">
      本软件以 MIT 协议开源，你可以自由使用、修改、合并、发布、分发、再许可和销售本软件的副本，无需任何授权。
    </el-text>
    <br><br>
  </div>
  <div style="position: relative;text-align: left;left: 20px">
    <el-text class="mx-1" type="primary">
      本软件按「原样」提供，不附带任何明示或暗示的保证。
    </el-text>
    <br><br><br>
  </div>

  <div style="position: relative;text-align:right">
    <el-tag class="ml-2" type="success">Copyright (C) 2026 infycat</el-tag>
    <br><br>
    <el-tag class="ml-2" type="success">https://github.com/infycat/McpNet</el-tag>
    <br><br><br>
  </div>
</template>

<script>

import {BrowserOpenURL} from "../../../wailsjs/runtime/runtime.js";

export default {
  data() {
    return {
      Port: ""
    }
  }, mounted() {

  }, methods: {
    callUrl(a) {
      BrowserOpenURL(a)
    }
  }
}
</script>