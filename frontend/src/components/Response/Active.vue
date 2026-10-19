<template>
  <div :style="`position: absolute; top: 0;left: 0;width: 100%;height: ${BodyRectHeight};`">
    <VueText ref="RawText" Language="'html'" Name="ActiveView"/>
  </div>
  <div :style="`position: absolute; top: ${RectHeight};display: flex;width: 100%;`">
    <el-select v-model="SendType" class="m-2" placeholder="Select" size="small" style="flex-grow: 0;width: 157px">
      <el-option
          v-for="item in SendTypeOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
      />
    </el-select>
    <el-select v-model="wsType" class="m-2" placeholder="Select" size="small" style="width: 122px;" v-show="IsWs">
      <el-option
          v-for="item in wsTypeOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
      />
    </el-select>
    <el-select v-model="direction" class="m-2" placeholder="Select" size="small" style="width: 115px;">
      <el-option
          v-for="item in directionOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
      />
    </el-select>
    <el-button size="small" style="flex-grow: 1;" @click="Send" v-show="IsSending===false">立即发送</el-button>
    <el-button size="small" style="flex-grow: 1;" disabled v-show="IsSending">正在发送</el-button>
  </div>

</template>

<script>
import VueText from "../Request/Text.vue";
import {CallGoDo, StrBase64Encode} from "../CallbackEventsOn.js";

export default {
  props: ['Height'],
  watch: {
    SendType(next, previous) {
      const editor = this.$refs.RawText
      if (!editor) return
      if (!this.updatingCode) {
        let content = editor.GetCode() || ''
        if (previous === 'HEX' && next === 'JSON') {
          const compactHex = content.replace(/0x/gi, '').replace(/\s+/g, '')
          if (/^(?:[0-9a-f]{2})+$/i.test(compactHex)) {
            const bytes = Uint8Array.from(compactHex.match(/.{2}/g), value => parseInt(value, 16))
            content = new TextDecoder('utf-8').decode(bytes)
          }
          try {
            content = JSON.stringify(JSON.parse(content), null, 2)
          } catch (error) {
            // Keep the current text editable so the user can repair it as JSON.
          }
          editor.SetCode(content)
        } else if (previous === 'JSON' && next === 'HEX') {
          try {
            const compactJson = JSON.stringify(JSON.parse(content))
            const bytes = new TextEncoder().encode(compactJson)
            content = Array.from(bytes, value => value.toString(16).padStart(2, '0')).join(' ').toUpperCase()
            editor.SetCode(content)
          } catch (error) {
            // Leave invalid JSON untouched instead of silently sending different data.
          }
        }
      }
      this.$nextTick(() => editor.SetLanguage(next === 'JSON' ? 'json' : 'plaintext'))
    },
    Height(value) {
      this._Height = value
    },
  },
  components: {VueText},
  computed: {
    BodyRectHeight() {
      console.log("BodyRectHeight",)
      return (parseInt(this._Height.replaceAll("px", "")) - 28) + "px"
    },
    RectHeight() {
      const top = (parseInt(this._Height.replaceAll("px", "")) - 25) + "px"
      return top
    },
  },
  data() {
    return {
      IsWs: false,
      IsSending: false,
      _Height: "0px",
      SendType: "JSON",
      updatingCode: false,
      direction: "Server",
      wsType: "Text",
      SendTypeOptions: [
        {
          value: 'HEX',
          label: '发送类型:HEX',
        },
        {
          value: 'Base64',
          label: '发送类型:Base64',
        },
        {
          value: 'GBK',
          label: '发送类型:String(GBK)',
        },
        {
          value: 'UTF8',
          label: '发送类型:String(UTF8)',
        },
        {
          value: 'JSON',
          label: '发送类型:JSON',
        },
      ],
      directionOptions: [
        {
          value: 'Client',
          label: '向 客户端 发送',
        },
        {
          value: 'Server',
          label: '向 服务器 发送',
        },
      ],
      wsTypeOptions: [
        {
          value: 'Text',
          label: '数据类型:Text',
        },
        {
          value: 'Binary',
          label: '数据类型:Binary',
        },
      ],
    }
  },
  methods: {
    SetCode(value) {
      if (value == null) {
        this.$refs.RawText.SetCode('')
        return
      }
      const bytes = value instanceof Uint8Array ? value : Uint8Array.from(value)
      const text = new TextDecoder('utf-8').decode(bytes)
      let formatted = text
      try {
        formatted = JSON.stringify(JSON.parse(text), null, 2)
        this.SendType = 'JSON'
      } catch (error) {
        this.SendType = 'HEX'
        formatted = Array.from(bytes, item => item.toString(16).padStart(2, '0')).join(' ').toUpperCase()
      }
      this.updatingCode = true
      this.$refs.RawText.SetCode(formatted)
      this.$nextTick(() => {
        this.$refs.RawText.SetLanguage(this.SendType === 'JSON' ? 'json' : 'plaintext')
        this.updatingCode = false
      })
    }, Send() {
      this.IsSending = true
      let content = StrBase64Encode(this.$refs.RawText.GetCode())
      let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
      const Data = {
        direction: this.direction,
        SendType: this.SendType,
        wsType: this.wsType,
        Data: content,
        Theology: window.Theology,
        IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
        IsTCP: way.indexOf("TCP") !== -1
      }
      CallGoDo("主动发送", Data).then(res => {
        this.IsSending = false
      })
    }
  },
  mounted() {
    this.$refs.RawText.SetReadOnly(false)
    this.$refs.RawText.SetLanguage('json')
  },
}
</script>
