<template>
  <div class="app-header-bar" @dblclick="clickWindowButton(2)" :style="backStyle">

    <div style="display: flex; align-items: center;" @dblclick.stop>
      <div style="cursor: pointer; display: flex; align-items: center;">

        <el-menu
            :default-active="activeIndex"
            class="el-menu-demo"
            mode="horizontal"
            :ellipsis="false"
            @select="handleSelect"
        >
          <el-sub-menu index="文件">
            <template #title>
              <span class="app-brand">
                <svg class="app-brand-mark" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
                  <rect x="2" y="3" width="20" height="5" rx="2" fill="currentColor"/>
                  <rect x="2" y="10" width="13" height="4" rx="2" fill="currentColor" opacity=".75"/>
                  <rect x="2" y="16" width="17" height="4" rx="2" fill="currentColor" opacity=".55"/>
                </svg>
                <span class="app-brand-text">McpNet</span>
              </span>
            </template>
            <el-menu-item index="打开文件" @click="OpenFile()" :disabled="Stop">打开文件</el-menu-item>
            <el-sub-menu index="保存文件" :disabled="Stop">
              <template #title>保存文件</template>
              <el-menu-item index="保存选中的文件" @click="SaveToFile(false)">保存选中的文件</el-menu-item>
              <el-menu-item index="保存全部" @click="SaveToFile(true)">保存全部</el-menu-item>
            </el-sub-menu>
          </el-sub-menu>

          <el-menu-item index="设置">
            <div style="display: flex; align-items: center;position: relative;top:0px">
              <div style="cursor: pointer; display: flex; align-items: center;">
                <el-tooltip class="item" effect="dark"
                            content="程序设置"
                            placement="top">
                  <el-icon>
                    <Setting/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>

          <el-menu-item index="清除全部数据" :disabled="Stop">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:0px">

                <el-tooltip class="item" effect="dark"
                            content="清空所有记录"
                            placement="top">
                  <el-icon>
                    <Delete/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>
          <el-menu-item index="清除全部过滤条件" :disabled="Stop">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:0px">

                <el-tooltip class="item" effect="dark"
                            content="清空全部过滤条件"
                            placement="top">
                  <el-icon>
                    <Filter/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>
          <el-menu-item index="全部放行">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:1px">
                <el-tooltip class="item" effect="dark"
                            content="全部放行"
                            placement="top">
                  <el-icon>
                    <CaretRight/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>
          <el-menu-item v-if="IsWindows" index="进程驱动">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:1px">
                <el-tooltip class="item" effect="dark"
                            content="进程驱动"
                            placement="top">
                  <el-icon>
                    <Monitor/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>
          <el-menu-item index="脚本编辑">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:1px">
                <el-tooltip class="item" effect="dark"
                            content="脚本编辑"
                            placement="top">
                  <el-icon>
                    <EditPen/>
                  </el-icon>
                </el-tooltip>

              </div>
            </div>
          </el-menu-item>
          <el-menu-item index="自动滚动">
            <div style="display: flex; align-items: center;">
              <div style="cursor: pointer; display: flex; align-items: center;position: relative;top:1px">
                <el-icon v-show="AutoRollShow===false">
                  <CircleCloseFilled/>
                </el-icon>
                <el-icon v-show="AutoRollShow">
                  <SuccessFilled/>
                </el-icon>
                <el-tooltip class="item" content="列表自动跟随显示" placement="top">
                  <span>自动滚动显示</span>
                </el-tooltip>
              </div>
            </div>
          </el-menu-item>
        </el-menu>
      </div>

    </div>
    <PartitionOperator @dblclick.stop/>
    <!-- 文本对比 -->
    <div style="display: flex; align-items: center;" @dblclick.stop>
      <div @click="clickTextCompare" style="cursor: pointer; display: flex; align-items: center;">
        <svg :class="svgStyle" xmlns="http://www.w3.org/2000/svg" width="14" height="16" viewBox="0 0 24 24">
          <polyline points="4 7 4 4 20 4 20 7" fill="none" stroke-linecap="round" stroke-linejoin="round"
                    stroke-width="2"/>
          <line x1="9" y1="20" x2="15" y2="20" fill="none" stroke-linecap="round" stroke-linejoin="round"
                stroke-width="2"/>
          <line x1="12" y1="4" x2="12" y2="20" fill="none" stroke-linecap="round" stroke-linejoin="round"
                stroke-width="2"/>
        </svg>
        <div style="width: 2px"></div>
        文本对比
      </div>
    </div>
    <PartitionOperator @dblclick.stop/>
    <!-- 证书安装教程 -->
    <div style="display: flex; align-items: center;" @dblclick.stop>
      <div @click="clickDocCompare" style="cursor: pointer; display: flex; align-items: center;">
        <el-icon>
          <Flag/>
        </el-icon>
        <div style="width: 2px"></div>
        证书安装教程
      </div>
    </div>
    <PartitionOperator @dblclick.stop/>
    <!-- 开源协议 -->
    <div style="display: flex; align-items: center;" @dblclick.stop>
      <div @click="clickOpenSourceProtocol" style="cursor: pointer; display: flex; align-items: center;">
        <el-icon>
          <Reading/>
        </el-icon>
        <div style="width: 2px"></div>
        开源协议
      </div>
    </div>


    <div style="position: absolute; right: 95px; cursor: pointer;z-index: 1000000;width: 16px;height: 16px"
         @dblclick.stop>
      <el-popover
          placement="top-start"
          :width="200"
          trigger="hover"
          popper-style="box-shadow: rgb(14 18 22 / 35%) 0px 10px 38px -10px, rgb(14 18 22 / 20%) 0px 10px 20px -15px; padding: 20px;"
      >
        <el-table :data="WayContent">
          <el-table-column width="150" property="ip" label="当前内网IP"/>
        </el-table>
        <template #reference>
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
            <path
                d="M24,0A24,24,0,1,0,48,24,24,24,0,0,0,24,0ZM35.32,35.8A40.15,40.15,0,0,0,37,25h9a21.91,21.91,0,0,1-5.85,13.94A24.94,24.94,0,0,0,35.32,35.8ZM8,39.1A21.92,21.92,0,0,1,2,25h9a40.05,40.05,0,0,0,1.71,10.94A25,25,0,0,0,8,39.1Zm4.64-26.81A40.21,40.21,0,0,0,11,23H2A21.91,21.91,0,0,1,7.85,9.08,25,25,0,0,0,12.66,12.28ZM25,15a24.92,24.92,0,0,0,8.51-1.85A38.76,38.76,0,0,1,35,23H25Zm0-2V2.1c3.2.61,6.05,4.1,7.88,9.12A22.9,22.9,0,0,1,25,13ZM23,2.1V13a22.91,22.91,0,0,1-7.88-1.74C17,6.19,19.8,2.71,23,2.1ZM23,15v8H13a38.75,38.75,0,0,1,1.48-9.87A24.93,24.93,0,0,0,23,15ZM13,25H23v8.2a24.9,24.9,0,0,0-8.44,1.89A38.63,38.63,0,0,1,13,25ZM23,35.23V45.9c-3.15-.6-6-4-7.8-8.9A22.89,22.89,0,0,1,23,35.23ZM25,45.9V35.22a22.93,22.93,0,0,1,7.85,1.66C31,41.85,28.18,45.3,25,45.9Zm0-12.7V25H35a38.7,38.7,0,0,1-1.51,10A24.94,24.94,0,0,0,25,33.2ZM37,23a40.21,40.21,0,0,0-1.64-10.72,24.94,24.94,0,0,0,4.8-3.21A21.91,21.91,0,0,1,46,23ZM38.71,7.66a23,23,0,0,1-4,2.71,21,21,0,0,0-4.5-7.48A22,22,0,0,1,38.71,7.66ZM13.3,10.36a23,23,0,0,1-4-2.71,22,22,0,0,1,8.52-4.76A21,21,0,0,0,13.3,10.36ZM9.47,40.5a23,23,0,0,1,3.92-2.65,20.82,20.82,0,0,0,4.42,7.25A22,22,0,0,1,9.47,40.5Zm25.2-2.79a23,23,0,0,1,4,2.65,22,22,0,0,1-8.5,4.75A21,21,0,0,0,34.67,37.71Z"
                fill="#0797E1"/>
          </svg>
        </template>
      </el-popover>


    </div>


    <div style="position: absolute; right: 60px; cursor: pointer;z-index: 1000000;font-size: 17px;top: 8px"
         @click="clickWindowButton(1)" @dblclick.stop>
      <el-icon>
        <SemiSelect/>
      </el-icon>
    </div>
    <div style="position: absolute; right: 33px; cursor: pointer;z-index: 1000000;font-size: 17px;top: 5px"
         @click="clickWindowButton(2)" @dblclick.stop>
      <el-icon v-if="Maximise===false">
        <TopRight/>
      </el-icon>
      <el-icon v-if="Maximise===true">
        <BottomLeft/>
      </el-icon>
    </div>
    <div style="position: absolute; right: 6px; cursor: pointer;z-index: 1000000;font-size: 17px;top: 5px"
         @click="clickWindowButton(3)" @dblclick.stop>
      <el-icon>
        <SwitchButton/>
      </el-icon>
      <!--       <span class="ag-icon ag-icon-cross ag-panel-title-bar-button-icon"></span>   -->
    </div>
  </div>
  <Strings ref="Strings" v-show="ShowSetting" :show="ShowSetting"   @keydown.stop="handleKeyDown"/>
  <Doc ref="Doc" v-show="ShowDocCompare" :show="ShowDocCompare" />
  <TextCompare ref="TextCompare" v-show="ShowTextCompare" :show="ShowTextCompare"   @keydown.stop="handleKeyDown"/>
  <OpenSourceProtocol ref="OpenSourceProtocol" v-show="ShowOpenSourceProtocol" :show="ShowOpenSourceProtocol" />
</template>

<script>
import AgOption from './option.vue';
import SwitchTheme from './SwitchTheme.vue';
import Strings from './Settings.vue';
import TextCompare from './TextCompare/TextCompare.vue';
import ComboBox from './ComboBox.vue';
import PartitionOperator from "./PartitionOperator.vue";
import '../../wailsjs/runtime/runtime.js';
import {EventsOn, WindowMinimise, WindowToggleMaximise} from "../../wailsjs/runtime/runtime.js";
import {Do} from "../../wailsjs/go/main/App.js";
import {CallGoDo, EventsDo, StrBase64Encode} from "./CallbackEventsOn.js";
import {CircleCloseFilled, SuccessFilled} from '@element-plus/icons-vue'
import {ElMessage} from "element-plus";
import Doc from "./CertDoc/Doc.vue";
import OpenSourceProtocol from "./OpenSourceProtocol/OpenSourceProtocol.vue";


export default {
  components: {
    OpenSourceProtocol,
    Doc,
    PartitionOperator,
    AgOption, SwitchTheme, Strings, TextCompare, ComboBox
  },
  computed: {
    IsWindows() {
      if (window.Theme) {
        return window.Theme.GOOS === "windows"
      }
      return false
    },
    backStyle() {
      let c = "height: 30px;background-color: var(--app-header, #202020);color:var(--app-text, #f5f5f5);border-bottom:1px solid var(--app-border, #424242);"
      return c + "--wails-draggable:drag"
    },
    svgStyle() {
      if (!this.theme) {
        return "dr2"
      } else {
        return "dr1"
      }
    },
    cvgStyle() {
      if (!this.theme) {
        return ""
      } else {
        return "fill: white;stroke: white"
      }
    },
    ShowSetting() {
      return this.UISettings
    },
    ShowTextCompare() {
      return this.TextCompare
    },
    ShowDocCompare() {
      return this.DocCompare
    },
    ShowOpenSourceProtocol() {
      return this.OpenSourceProtocol
    },
  },
  data() {
    return {
      Stop: false,
      AutoRollShow: false,
      WayContent: [
        {
          ip: '暂未获取到',
        },
      ],
      activeIndex: "打开文件",
      Maximise: false,
      get theme() {
        return window.Theme.IsDark
      },
      set theme(newValue) {
        window.Theme.IsDark = newValue
      },
      get UISettings() {
        return window.UI.Settings
      },
      set UISettings(newValue) {
        window.UI.Settings = newValue
      },
      get TextCompare() {
        return window.UI.TextCompare
      },
      set TextCompare(newValue) {
        window.UI.TextCompare = newValue
      },
      get DocCompare() {
        return window.UI.DocCompare
      },
      set DocCompare(newValue) {
        window.UI.DocCompare = newValue
      },
      get OpenSourceProtocol() {
        return window.UI.OpenSourceProtocol
      },
      set OpenSourceProtocol(newValue) {
        window.UI.OpenSourceProtocol = newValue
      },
      WindowSize: {min: 0, max: 0},
    }
  },
  methods: {
    ShowScriptEditing() {
      this.UISettings = false
      CallGoDo("加载配置", null).then(res => {
        this.$nextTick(() => {
          this.UISettings = true
          window.SetUILevel("Settings")
          window.vm.Settings.activeName = "脚本编辑"
        })
      })
    },
    ShowDrive() {
      this.UISettings = false
      CallGoDo("加载配置", null).then(res => {
        this.$nextTick(() => {
          this.UISettings = true
          window.SetUILevel("Settings")
          window.vm.Settings.activeName = "进程拦截"
        })
      })
    },
    rollShow() {
      this.AutoRollShow = !this.AutoRollShow
      window.vm.List.ListFollowShow = this.AutoRollShow
    },
    SetAutoRollShow(a) {
      this.AutoRollShow = a
    },
    handleSelect(key, path) {
      this.activeIndex = ""
      if (key === "设置") {
        this.clickSettings()
        return
      }
      // 清除全部数据
      if (key === "清除全部数据") {
        this.clickRemoveAll(1)
        return
      }
      // 清除全部过滤条件
      if (key === "清除全部过滤条件") {
        this.clickRemoveAll(2)
        return
      }
      //全部放行
      if (key === "全部放行") {
        this.ReleaseAll()
        return
      }
      //进程驱动
      if (key === "进程驱动") {
        this.ShowDrive()
        return
      }

      //脚本编辑
      if (key === "脚本编辑") {
        this.ShowScriptEditing()
        return
      }
      //自动滚动
      if (key === "自动滚动") {
        this.rollShow()
        return
      }


      console.log(key, path, this.activeIndex, "ok")

    },
    clickWindowButton(index) {
      if (index === 1) {
        WindowMinimise()
      } else if (index === 2) {
        let tl = window.vm.List.getTools();
        if (tl) {
          const w = parseInt(tl.style.width.replace('px', ''))

          if (this.Maximise) {
            this.WindowSize.max = w;
          } else {
            this.WindowSize.min = w;
          }
        }
        this.Maximise = !this.Maximise
        WindowToggleMaximise()
        this.$nextTick(() => {
          if (this.Maximise) {
            if (this.WindowSize.max < 30) {
              tl.style.width = '300px'
            } else {
              tl.style.width = this.WindowSize.max + 'px'
            }
          } else {
            tl.style.width = this.WindowSize.min + 'px'
          }

        })
      } else if (index === 3) {
        const objs = {}
        const filterModel = window.vm.List.agGridApi.getFilterModel();
        // 遍历过滤器模型，获取每个列的过滤器信息
        for (const colId in filterModel) {
          objs[colId] = filterModel[colId]
        }
        CallGoDo("CloseWindow", {
          Filter: StrBase64Encode(JSON.stringify(objs)),
          KeysStrings: StrBase64Encode(JSON.stringify(window.KeysStrings)),
          StorageColumns: StrBase64Encode(JSON.stringify(window.vm.List.$refs.agGrid.gridOptions.columnApi.getColumnState()))
        })
      }

    },
    clickSettings() {
      this.UISettings = false
      CallGoDo("加载配置", null).then(res => {
        this.$nextTick(() => {
          this.UISettings = true
          window.SetUILevel("Settings")
          window.vm.Settings.activeName = ""
        })
      })
    },
    clickTextCompare() {
      this.TextCompare = false
      this.$nextTick(() => {
        this.TextCompare = true
        window.SetUILevel("TextComparison")
      })
    },
    clickDocCompare() {
      this.DocCompare = false
      this.$nextTick(() => {
        this.DocCompare = true
        window.SetUILevel("DocCompare")
      })
    },
    clickOpenSourceProtocol() {
      this.OpenSourceProtocol = false
      this.$nextTick(() => {
        this.OpenSourceProtocol = true
        window.SetUILevel("OpenSourceProtocol")
      })
    },
    SaveToFile(ALL) {
      const obj = {
        Title: "请选择文件保存位置",
        Filters: [
          {Name: "McpNet抓包文件", Pattern: "*.syn"}
        ]
      }
      CallGoDo("保存文件对话框", obj).then(res => {
        if (res !== '') {
          const array = []
          if (!ALL) {
            for (let i = 0; i < window.vm.List.agSelectedArray.length; i++) {
              array.push(window.vm.List.agSelectedArray[i].data['Theology'])
            }
          }
          this.Stop = true
          CallGoDo("保存文件", {Path: res, ALL: ALL, Data: array}).then(res => {
            this.Stop = false
            if (res) {
              ElMessage({
                message: "文件已储存",
                type: 'success',
              })
            } else {
              ElMessage({
                message: "保存文件失败",
                dangerouslyUseHTMLString: true,
                type: 'error',
              })
            }
          })
        }
      })
    },
    OpenFile() {
      const obj = {
        Title: "请选择抓包记录文件",
        Filters: [
          {Name: "McpNet抓包文件", Pattern: "*.syn"},
        ]
      }
      CallGoDo("选择文件", obj).then(res => {
        if (res !== '') {
          this.Stop = true
          CallGoDo("打开记录文件", {Path: res}).then(res => {
            this.Stop = false
            if (res) {
              ElMessage({
                message: "文件已载入",
                type: 'success',
              })
            } else {
              ElMessage({
                message: "载入文件失败",
                dangerouslyUseHTMLString: true,
                type: 'error',
              })
            }
          })
        }
      })
    },
    ReleaseAll() {
      CallGoDo("全部放行", null)
    },
    clickRemoveAll(mode) {
      if (mode === 2) {
        this.$nextTick(() => {
          window.vm.List.agGridApi.setFilterModel([]); // 清空过滤器条件
          const ok = Object.keys(window.vm.List.RowDataHashMap).length < 1
          if (ok) {
            window.vm.List.RowData = [{"序号": 1001, "ico": "error"}]
            window.vm.List.RowDataHashMap = {}
            window.vm.List.agSelectedLine = null
            window.vm.List.agGridApi.setRowData(window.vm.List.RowData);
            window.vm.List.index = 0
          }
          this.$nextTick(() => {
            window.vm.List.RefreshRenderedNodes();
            if (ok) {
              window.vm.List.RowData = []
              window.vm.List.RowDataHashMap = {}
              window.vm.List.agSelectedLine = null
              window.vm.List.agGridApi.setRowData(window.vm.List.RowData);
              window.vm.List.index = 0
            }
            const columnFilter = window.vm.List.agGridApi.getFilterInstance('响应长度');
            columnFilter.setModel({
              type: 'notContains',
              filter: '0/0'
            });
            window.vm.List.agGridApi.onFilterChanged();
          })
        })
        return
      }
      CallGoDo("清空", null).then(res => {
        window.vm.List.RowData = []
        window.vm.List.RowDataHashMap = {}
        window.vm.List.agSelectedLine = null
        window.vm.List.agGridApi.setRowData(window.vm.List.RowData);
        window.vm.List.index = 0

      })
    },
    handleKeyDown(event) {
      event.stopPropagation();
    }
  },
  mounted() {
    try {
      EventsOn("Do", (Request) => {
            EventsDo(Request)
          },
      )
    } catch (e) {
      console.log("UpdateList error", e)
    }
    window.vm.Header = this

    this.$nextTick(() => {
      CallGoDo("获取内网IP", null).then(res => {
        const objs = []
        for (let i = 0; i < res.length; i++) {
          if (res[i].startsWith("169.")) {
            continue
          }
          objs.push({
            ip: res[i],
          })
        }
        if (res.length < 1) {
          this.WayContent = [{ip: "未获取到"}]
          return
        }
        this.WayContent = objs
      })

    })
  }
}
</script>
<style scoped>
.app-brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--app-text, #f5f5f5);
  font-weight: 600;
  font-size: 14px;
  letter-spacing: .3px;
}

.app-brand-mark {
  flex: 0 0 16px;
  color: var(--app-accent, #409eff);
}

.app-brand-text {
  line-height: 1;
}

/* Keep the app toolbar independent from AG Grid's header-cell flex rules. */
.app-header-bar {
  box-sizing: border-box;
  display: flex;
  flex: 0 0 30px;
  align-items: center;
  width: 100%;
  min-width: 0;
  height: 30px;
  min-height: 30px;
  overflow: hidden;
}

/*深色模式显示*/
.dr1 {
  fill: #cccccc;
  stroke: #cccccc
}

/*浅色模式显示*/
.dr2 {
  fill: #1e1d1d;
  stroke: #1e1d1d
}

.el-menu-item {
  padding: 5px
}
</style>
