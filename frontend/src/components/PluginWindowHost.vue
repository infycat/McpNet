<template>
  <div v-if="visible" class="plugin-window-overlay" @click.self="close" @keydown.esc="close">
    <section class="plugin-window" role="dialog" aria-modal="true" :aria-label="title">
      <header class="plugin-window__header">
        <div class="plugin-window__title">{{ title }}</div>
        <button class="plugin-window__close" type="button" :aria-label="closeLabel" @click="close">×</button>
      </header>
      <iframe
        ref="frame"
        class="plugin-window__frame"
        :title="title"
        sandbox="allow-scripts"
        :srcdoc="srcdoc"
      />
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { CallGoDo } from './CallbackEventsOn.js'

const visible = ref(false)
const title = ref('Plugin window')
const pluginId = ref('')
const srcdoc = ref('')
const frame = ref(null)
const closeLabel = computed(() => {
  try {
    return localStorage.getItem('sunny_lang') === 'tw' ? '關閉' : 'Close'
  } catch (_) {
    return 'Close'
  }
})

function handleOpen(event) {
  const detail = event && event.detail
  if (!detail || typeof detail.Id !== 'string' || typeof detail.HTML !== 'string') return
  if (!detail.Id || !detail.HTML.trim() || detail.HTML.length > 256 * 1024) return
  pluginId.value = detail.Id
  title.value = typeof detail.Title === 'string' && detail.Title.trim()
    ? detail.Title.slice(0, 128)
    : detail.Id
  const csp = '<meta http-equiv="Content-Security-Policy" content="default-src \'none\'; img-src data: blob:; style-src \'unsafe-inline\'; script-src \'unsafe-inline\'; connect-src \'none\'; form-action \'none\'; navigate-to \'none\'">'
  const html = detail.HTML
  const doctype = html.match(/^\s*<!doctype[^>]*>/i)
  srcdoc.value = doctype ? doctype[0] + csp + html.slice(doctype[0].length) : csp + html
  visible.value = true
}

function close() {
  visible.value = false
  srcdoc.value = ''
  pluginId.value = ''
}

async function handleFrameMessage(event) {
  if (!frame.value?.contentWindow || event.source !== frame.value.contentWindow) return
  const data = event.data
  if (!data || data.type !== 'mcpnet-plugin-action') return
  const action = String(data.action || '')
  if (!action || action.length > 128 || !pluginId.value) return
  let payload
  try {
    payload = JSON.stringify(data.payload == null ? null : data.payload)
  } catch (_) {
    return
  }
  if (payload.length > 16 * 1024) return
  try {
    const result = await CallGoDo('插件界面動作', { Id: pluginId.value, Action: action, Payload: payload })
    if (frame.value?.contentWindow) {
      frame.value.contentWindow.postMessage({
        type: 'mcpnet-plugin-result',
        action,
        result: result && result.result ? result.result : '',
        error: result && result.error ? result.error : '',
      }, '*')
    }
  } catch (error) {
    if (frame.value?.contentWindow) {
      frame.value.contentWindow.postMessage({
        type: 'mcpnet-plugin-result',
        action,
        error: String(error?.message || error),
      }, '*')
    }
  }
}

onMounted(() => {
  window.addEventListener('mcpnet-plugin-open-window', handleOpen)
  window.addEventListener('message', handleFrameMessage)
})

onBeforeUnmount(() => {
  window.removeEventListener('mcpnet-plugin-open-window', handleOpen)
  window.removeEventListener('message', handleFrameMessage)
})
</script>

<style scoped>
.plugin-window-overlay {
  position: fixed;
  inset: 0;
  z-index: 200000;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  padding: 20px;
  background: rgb(0 0 0 / 62%);
}

.plugin-window {
  display: flex;
  flex-direction: column;
  width: min(1000px, 96vw);
  height: min(760px, 90vh);
  overflow: hidden;
  color: var(--app-text, #d4d4d4);
  background: var(--app-surface, #1b1b1b);
  border: 1px solid var(--app-border, #3c3c3c);
  border-radius: 6px;
  box-shadow: 0 12px 40px rgb(0 0 0 / 55%);
}

.plugin-window__header {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  padding: 0 12px 0 16px;
  background: var(--app-header, #202020);
  border-bottom: 1px solid var(--app-border, #333);
}

.plugin-window__title {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plugin-window__close {
  width: 34px;
  height: 34px;
  color: inherit;
  font: inherit;
  font-size: 22px;
  background: transparent;
  border: 0;
  border-radius: 4px;
  cursor: pointer;
}

.plugin-window__close:hover,
.plugin-window__close:focus-visible {
  background: rgb(255 255 255 / 12%);
  outline: 2px solid var(--app-accent, #409eff);
}

.plugin-window__frame {
  flex: 1 1 auto;
  width: 100%;
  min-height: 0;
  background: var(--app-surface, #1b1b1b);
  border: 0;
}

@media (max-width: 640px) {
  .plugin-window-overlay { padding: 8px; }
  .plugin-window { width: 100%; height: 94vh; }
}
</style>
