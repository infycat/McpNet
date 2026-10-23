<script>
import Home from './components/Home.vue'
import PluginWindowHost from './components/PluginWindowHost.vue'
import {CallGoDo} from "./components/CallbackEventsOn.js";
import {ElNotification} from "element-plus";
import {LogError, LogInfo} from "../wailsjs/runtime/runtime";

window.vm = {
  Footer: null,
  List: null,
  Header: null,
  Theme: null,
  Tabs: {
    Request: null,
    Response: null,
    ToolPanel: null
  },
  ListColorManager: null,
  Find: null,
  Settings: null,
  OpenSourceProtocol: null,
  Doc: null,
  CloseWindow: null,
  IEProxy: null,
  ListenerState: null,
}
window.VsCodeEdit = {JavaScriptEdit: false}
let Interval = 0;
export default {
  components: {
    Home,
    PluginWindowHost
  },
  data() {
    return {}
  },
  mounted() {
    LogInfo('Vue root component mounted');
    Interval = setInterval(function () {
      let ready = true
      for (const fieldName in window.VsCodeEdit) {
        if (window.VsCodeEdit[fieldName] !== true) {
          ready = false
          break
        }
      }
      if (!ready) return
      clearInterval(Interval)
      CallGoDo("init", null).then(() => {
        LogInfo('Frontend initialization completed');
        ElNotification({
          title: '免责申明',
          dangerouslyUseHTMLString: true,
          duration: 10000,
          position: "bottom-right",
          offset: 25,
          message: '<span  style="color: red">本工具仅限 学术交流 技术探讨使用</span><br><span style="color: red">禁止用于一切非法用途</span><br><span style="color: red">否则造成的一切后果自负</span>',
        })
        setTimeout(() => {
          ElNotification({
            title: '证书安装',
            dangerouslyUseHTMLString: true,
            duration: 5000,
            position: "bottom-right",
            offset: 25,
            message: '<span  style="color: #3ccbff">若是第一次使用,请查看"证书安装教程"</span>',
          })
        }, 1000)
      }).catch((error) => {
        console.error('McpNet initialization failed', error)
        LogError(`Frontend initialization failed: ${error?.stack || error}`)
        ElNotification({
          title: '啟動失敗 / Startup failed',
          message: '應用程式初始化失敗，請查看 %LOCALAPPDATA%\\McpNet_new\\logs\\McpNet_startup.log。',
          type: 'error',
          duration: 0,
        })
      })
    }, 200)
  }
}


document.addEventListener('mousemove', event => {
  const mouseX = event.clientX;
  const mouseY = event.clientY;
  const as = document.querySelector(".shadow-root-host")
  if (as) {
    const cnm = as.shadowRoot.querySelectorAll(".actions-container")
    if (cnm.length < 1) {
      return
    }
    const menu = cnm[0].childNodes
    for (let i = 0; i < menu.length; i++) {
      const rect = menu[i].getBoundingClientRect();
      const x = rect.left;
      const y = rect.top;
      const width = rect.width;
      const height = rect.height;
      const c = menu[i].getElementsByClassName("action-menu-item")
      if ((mouseX >= x && mouseX < x + width) && (mouseY >= y && mouseY < y + height)) {
        menu[i].classList.add('focused');
        if (c.length > 0) {
          c[0].style = 'color: var(--vscode-menu-selectionForeground); background-color: var(--vscode-menu-selectionBackground); outline: 1px solid var(--vscode-menu-selectionBorder); outline-offset: -1px;'
        }
      } else {
        if (c.length > 0) {
          c[0].style = 'color: var(--vscode-menu-foreground);'
        }
        menu[i].classList.remove('focused');
      }
    }
  }
  const el = document.getElementsByClassName("is-active")
  if (el.length > 0) {
    for (let i = 0; i < el.length; i++) {
      const element = el[i]
      if (element && element.classList) {
        const ClassName = Array.from(element.classList).join(" ")
        if (ClassName.indexOf("el-menu-") !== -1 || ClassName.indexOf("el-sub-menu") !== -1) {
          element.classList.remove("is-active")
        }
      }
    }
  }
});

</script>

<template>
  <Home v-no-scroll/>
  <PluginWindowHost />
</template>


<style scoped>
/*禁止元素文本被选择 */
.no-select {
  user-select: none;
}
</style>
