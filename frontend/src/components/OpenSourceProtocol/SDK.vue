<template>
  <div class="source-community">
    <section class="source-section" aria-labelledby="source-heading">
      <h3 id="source-heading">軟件源碼、SDK 與文件</h3>
      <el-link
        class="source-link"
        type="warning"
        :href="sourceUrl"
        :underline="false"
        @click.prevent="callUrl(sourceUrl)"
      >
        {{ sourceUrl }}
      </el-link>
    </section>

    <el-divider />

    <section class="source-section" aria-labelledby="community-heading">
      <h3 id="community-heading">社群交流</h3>
      <el-link
        class="source-link"
        type="primary"
        :href="discordUrl"
        :underline="false"
        @click.prevent="callUrl(discordUrl)"
      >
        加入 Discord 社群 · discord.gg/b49ZA7qvbr
      </el-link>
    </section>
  </div>
</template>

<script>
import {BrowserOpenURL} from "../../../wailsjs/runtime/runtime.js";

export default {
  data() {
    return {
      sourceUrl: "https://github.com/infycat/McpNet",
      discordUrl: "https://discord.gg/b49ZA7qvbr"
    }
  },
  methods: {
    callUrl(a) {
      BrowserOpenURL(a)
    }
  }
}
</script>

<style scoped>
.source-community {
  max-width: 620px;
  margin: 0 auto;
  padding: 24px 20px;
  text-align: center;
}

.source-section h3 {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
}

.source-link {
  min-height: 40px;
  max-width: 100%;
  padding: 0 14px;
  border-radius: 6px;
  overflow-wrap: anywhere;
}

.source-community :deep(.el-divider) {
  margin: 24px 0;
}
</style>
