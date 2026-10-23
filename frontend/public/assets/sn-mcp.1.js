/* McpNet MCP + language settings. Injected as Element-Plus-styled collapse items. */
(function () {
  var WRAP_ID = 'sn-extra-settings';
  var MCP_ID = 'sn-mcp-panel';
  var state = { info: null, busy: false, scrollDone: false, processScrolled: false };
  window.__snMcp = window.__snMcp || { runs: 0, err: '' };
  window.addEventListener('error', function (e) { window.__snMcp.err = String(e.message || e); });

  function currentLang() {
    try { return localStorage.getItem('sunny_lang') || 'zh-CN'; } catch (e) { return 'zh-CN'; }
  }
  function L(cn, tw, en) {
    var l = currentLang();
    return l === 'en' ? en : (l === 'tw' ? tw : cn);
  }
  function doCommand(command, args) {
    if (!window.go || !window.go.main || !window.go.main.App || !window.go.main.App.Do) {
      return Promise.reject(new Error('Wails bridge not ready'));
    }
    return window.go.main.App.Do({ Command: command, Args: args == null ? null : args });
  }
  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }
  function svg(pathD, size) {
    var ns = 'http://www.w3.org/2000/svg';
    var s = document.createElementNS(ns, 'svg');
    s.setAttribute('viewBox', '0 0 24 24');
    s.setAttribute('width', size || '16');
    s.setAttribute('height', size || '16');
    s.setAttribute('fill', 'none');
    s.setAttribute('stroke', 'currentColor');
    s.setAttribute('stroke-width', '1.8');
    s.setAttribute('stroke-linecap', 'round');
    s.setAttribute('stroke-linejoin', 'round');
    var p = document.createElementNS(ns, 'path');
    p.setAttribute('d', pathD);
    s.appendChild(p);
    return s;
  }
  function chevron() {
    var i = el('i', 'el-collapse-item__arrow');
    i.appendChild(svg('M9 6l6 6-6 6', '14'));
    return i;
  }
  function collapseItem(iconPath, title, extra, content, open) {
    var item = el('div', 'el-collapse-item sn-extra-item');
    var head = el('div', 'el-collapse-item__header');
    head.setAttribute('role', 'button');
    head.tabIndex = 0;
    var ic = el('span', 'sn-extra-icon');
    ic.style.marginRight = '8px'; ic.style.display = 'inline-flex';
    ic.appendChild(svg(iconPath, '16'));
    head.appendChild(ic);
    head.appendChild(el('span', 'sn-extra-title', title));
    if (extra) head.appendChild(extra);
    var arrow = chevron();
    arrow.style.marginLeft = 'auto'; arrow.style.color = '#aab7c2';
    head.appendChild(arrow);
    var wrap = el('div', 'el-collapse-item__wrap');
    var inner = el('div', 'el-collapse-item__content');
    inner.style.width = '100%';
    inner.style.maxWidth = '100%';
    inner.style.boxSizing = 'border-box';
    inner.style.textAlign = 'left';
    inner.appendChild(content);
    wrap.appendChild(inner);
    item.appendChild(head);
    item.appendChild(wrap);

    function setOpen(v) {
      item.classList.toggle('is-active', v);
      wrap.style.display = v ? '' : 'none';
    }
    setOpen(!!open);
    head.addEventListener('click', function () {
      setOpen(!item.classList.contains('is-active'));
    });
    head.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); head.click(); }
    });
    return item;
  }
  function button(text, small) {
    var b = el('button', 'el-button el-button--default' + (small ? ' el-button--small' : ''), text);
    b.type = 'button';
    b.style.marginLeft = '6px';
    return b;
  }
  function input(value, placeholder) {
    var i = el('input', 'el-input__inner');
    i.value = value || '';
    i.placeholder = placeholder || '';
    i.style.height = '30px';
    i.style.flex = '1';
    i.style.minWidth = '0';
    i.style.boxSizing = 'border-box';
    return i;
  }
  function textarea() {
    var t = el('textarea', 'el-textarea__inner');
    t.readOnly = true;
    t.style.height = '132px';
    t.style.width = '100%';
    t.style.maxWidth = '100%';
    t.style.boxSizing = 'border-box';
    t.style.fontFamily = 'Consolas, monospace';
    t.style.fontSize = '12px';
    return t;
  }
  function row(children) {
    var d = el('div', 'sn-extra-row');
    d.style.display = 'flex';
    d.style.flexWrap = 'wrap';
    d.style.minWidth = '0';
    d.style.alignItems = 'center';
    d.style.gap = '8px';
    d.style.marginBottom = '10px';
    children.forEach(function (c) { d.appendChild(c); });
    return d;
  }
  function render(panel) {
    var info = state.info || {};
    var running = !!info.running;
    var st = panel.querySelector('.sn-mcp-state');
    if (st) {
      st.textContent = running ? L('运行中', '運行中', 'Running') : L('未运行', '未運行', 'Stopped');
      st.style.color = running ? '#67e8a0' : '#ff9b9b';
      st.title = info.error || '';
    }
    var url = panel.querySelector('.sn-mcp-url');
    if (url) url.textContent = info.url || L('未启动', '未啟動', 'Not started');
    var path = panel.querySelector('.sn-mcp-path');
    if (path && document.activeElement !== path) path.value = info.path || '';
    var out = panel.querySelector('.sn-mcp-json');
    if (out) out.value = info.json || '';
    var err = panel.querySelector('.sn-mcp-error');
    if (err) {
      err.textContent = info.error || '';
      err.style.display = info.error ? 'block' : 'none';
    }
  }
  function refresh(panel) {
    if (!panel || state.busy) return;
    doCommand('mcp信息', null).then(function (info) {
      state.info = info || {};
      render(panel);
    }).catch(function (e) {
      var err = panel.querySelector('.sn-mcp-error');
      if (err) { err.textContent = String(e); err.style.display = 'block'; }
    });
  }
  function buildLanguageItem() {
    var sel = el('select', 'el-input__inner');
    sel.style.height = '30px';
    sel.style.width = '220px';
    [['zh-CN', '简体中文'], ['tw', '繁體中文'], ['en', 'English']].forEach(function (p) {
      var o = el('option', '', p[1]); o.value = p[0]; sel.appendChild(o);
    });
    sel.value = currentLang();
    sel.onchange = function () {
      try { localStorage.setItem('sunny_lang', sel.value); } catch (e) {}
      location.reload();
    };
    var content = el('div');
    content.appendChild(row([el('span', '', L('界面语言', '介面語言', 'UI language')), sel]));
    return collapseItem('M12 3a9 9 0 100 18 9 9 0 000-18zM3 12h18M12 3c2.5 2.7 4 6 4 9s-1.5 6.3-4 9c-2.5-2.7-4-6-4-9s1.5-6.3 4-9z',
      L('语言', '語言', 'Language'), null, content, false);
  }
  function buildMcpItem() {
    var panel = el('div', 'sn-mcp-body');
    panel.style.width = '100%'; panel.style.maxWidth = '760px'; panel.style.textAlign = 'left'; panel.style.boxSizing = 'border-box'; panel.style.overflow = 'hidden';
    panel.id = MCP_ID;
    var statusWrap = el('span', 'sn-mcp-state');
    statusWrap.textContent = L('未运行', '未運行', 'Stopped');
    statusWrap.style.fontSize = '12px';
    statusWrap.style.marginLeft = '10px';

    var url = el('code', 'sn-mcp-url');
    url.style.marginLeft = 'auto';
    url.style.maxWidth = '100%';
    url.style.overflow = 'hidden';
    url.style.textOverflow = 'ellipsis';
    url.style.wordBreak = 'break-all';
    url.style.fontSize = '12px';
    url.style.color = '#9fd3ff';
    var stateRow = el('div');
    stateRow.style.display = 'flex';
    stateRow.style.flexWrap = 'wrap';
    stateRow.style.gap = '6px';
    stateRow.style.width = '100%';
    stateRow.style.boxSizing = 'border-box';
    stateRow.style.alignItems = 'center';
    stateRow.appendChild(statusWrap);
    stateRow.appendChild(url);
    stateRow.style.marginBottom = '10px';
    panel.appendChild(stateRow);

    var path = input('', L('留空使用默认路径', '留空使用預設路徑', 'Empty = default'));
    path.className += ' sn-mcp-path';
    var save = button(L('保存路径', '儲存路徑', 'Save path'), true);
    var restart = button(L('重启', '重啟', 'Restart'), true);
    panel.appendChild(el('div', '', L('配置文件', '設定檔', 'Config')));
    path.style.width = '100%';
    path.style.margin = '4px 0 8px';
    panel.appendChild(path);
    panel.appendChild(row([save, restart]));
    var note = el('div', '', L('若客户端连接失败：确认 McpNet 正在运行、2026 端口未被占用，再点“重启”。', '若客戶端連線失敗：確認 McpNet 正在執行、2026 連接埠未被占用，再點「重啟」。', 'If the client cannot connect: keep McpNet running, free port 2026, then click Restart.'));
    note.className = 'sn-mcp-note';
    note.style.fontSize = '12px';
    note.style.color = '#aab7c2';
    note.style.marginBottom = '8px';
    panel.appendChild(note);

    var permissionNote = el('div', '', L(
      'AI 可透過 MCP 查看封包中繼資料、協助建立／導入插件，並操作既有連線；讀取封包本文、發送資料、啟用插件或更改監聽／系統代理前，程式會逐次詢問你。只連接可信任的 AI 客戶端。',
      'AI 可透過 MCP 查看封包中繼資料、協助建立／匯入插件，並操作既有連線；讀取封包本文、發送資料、啟用插件或更改監聽／系統代理前，程式會逐次詢問你。只連接可信任的 AI 客戶端。',
      'AI clients can inspect capture metadata, help create/import plugins, and operate existing sessions through MCP. The app asks you before sharing payloads, sending data, enabling a plugin, or changing capture/proxy settings. Connect trusted AI clients only.'
    ));
    permissionNote.style.cssText = 'font-size:12px;line-height:1.6;color:var(--app-muted-text,#aab7c2);margin:0 0 10px;';
    panel.appendChild(permissionNote);

    var out = textarea();
    out.className += ' sn-mcp-json';
    panel.appendChild(out);

    var copy = button(L('复制 JSON', '複製 JSON', 'Copy JSON'), true);
    var hint = el('span', '', L('复制这段 JSON 到 AI 客户端的 MCP 配置中', '複製這段 JSON 到 AI 客戶端的 MCP 設定中', 'Paste this JSON into your AI client MCP config'));
    hint.style.fontSize = '12px';
    hint.style.color = '#aab7c2';
    panel.appendChild(row([copy, hint]));

    var err = el('div', 'sn-mcp-error');
    err.style.display = 'none';
    err.style.color = '#ff9b9b';
    err.style.fontSize = '12px';
    err.style.whiteSpace = 'pre-wrap';
    panel.appendChild(err);

    save.onclick = function () {
      state.busy = true;
      doCommand('mcp设置路径', { Path: path.value.trim() }).then(function (info) {
        state.info = info || {};
        if (info && info.error) throw new Error(info.error);
        render(panel);
      }).catch(function (e) {
        err.textContent = String(e); err.style.display = 'block';
      }).then(function () { state.busy = false; });
    };
    restart.onclick = function () {
      state.busy = true;
      doCommand('mcp重启', null).then(function (info) {
        state.info = info || {};
        render(panel);
      }).catch(function (e) {
        err.textContent = String(e); err.style.display = 'block';
      }).then(function () { state.busy = false; });
    };
    copy.onclick = function () {
      var text = out.value || '';
      var done = function () {
        copy.textContent = L('已复制', '已複製', 'Copied');
        setTimeout(function () { copy.textContent = L('复制 JSON', '複製 JSON', 'Copy JSON'); }, 1500);
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(function () {
          out.select(); document.execCommand('copy'); done();
        });
      } else {
        out.select(); document.execCommand('copy'); done();
      }
    };
    var item = collapseItem('M4 6h16M4 12h16M4 18h16',
      'MCP', null, panel, false);
    return item;
  }
  function maybeScrollProcess(dlg) {
    if (!dlg || dlg.offsetParent === null || state.processScrolled) return;
    var title = dlg.querySelector('.ag-panel-title-bar-title');
    var t = title ? (title.textContent || '') : '';
    if (!/进程|進程|Process/i.test(t)) return;
    var items = dlg.querySelectorAll('.el-collapse-item');
    var target = null;
    for (var i = 0; i < items.length; i++) {
      var h = items[i].querySelector('.el-collapse-item__header');
      var ht = h ? (h.textContent || '') : '';
      if (/进程拦截|進程攔截|Process Interception/i.test(ht)) { target = items[i]; break; }
    }
    if (!target) return;
    state.processScrolled = true;
    var head = target.querySelector('.el-collapse-item__header');
    if (head && !target.classList.contains('is-active')) head.click();
    setTimeout(function () {
      try { target.scrollIntoView({ block: 'center', inline: 'nearest' }); } catch (e) {}
      var p = target.parentElement;
      while (p && p !== document.body) {
        if (p.scrollHeight > p.clientHeight + 1) {
          p.scrollTop = Math.max(0, target.offsetTop - p.offsetTop - 40);
          break;
        }
        p = p.parentElement;
      }
    }, 80);
  }
  var PLUGIN_DOCS = [
    ['package main', ''],
    ['', '// 插件入口：按需实现任意一个或多个'],
    ['func OnHTTPRequest()  {}', '// 请求发出前'],
    ['func OnHTTPResponse() {}', '// 回包到达后'],
    ['func OnWebSocket()    {}', '// WS 帧收发'],
    ['func OnUDP()          {}', '// UDP 数据'],
    ['func OnPluginAction() {}', '// 內置窗口回調動作'],
    ['', ''],
    ['// ---- 上下文 ----', ''],
    ['mcpnet.Direction() string', '// "in" 接收 / "out" 发送'],
    ['mcpnet.Action() string', '// UI 動作名稱'],
    ['mcpnet.Payload() string', '// UI 傳入的 JSON 字串'],
    ['mcpnet.SetUIResult(s string)', '// 回傳至插件窗口'],
    ['mcpnet.Host() string', '// 主机名'],
    ['mcpnet.Path() string', '// 路径'],
    ['mcpnet.URL() string', '// 完整 URL'],
    ['mcpnet.Method() string', '// 请求方法'],
    ['mcpnet.PID() int', '// 进程 PID'],
    ['mcpnet.Process() string', '// 进程名'],
    ['mcpnet.Body() []byte', '// 当前数据'],
    ['mcpnet.SetBody(b []byte)', '// 修改当前挂钩的数据'],
    ['mcpnet.SetWireBody(b []byte)', '// 明确要求改写实际发送的数据'],
    ['mcpnet.DisplayBody() []byte', '// 读取仅供界面显示的数据'],
    ['mcpnet.SetDisplayBody(b []byte)', '// 设置界面显示，不改动线上封包'],
    ['mcpnet.OpenWindow(title, html string) bool', '// Go hook 建立內置插件窗口'],
    ['mcpnet.OpenExternalURL(url string) bool', '// Go hook 開啟外部 HTTP(S) 頁面'],
    ['', ''],
    ['// ---- 工具 ----', ''],
    ['mcpnet.HexEncode(b []byte) string', ''],
    ['mcpnet.HexDecode(s string) []byte', ''],
    ['mcpnet.Base64Encode(b []byte) string', ''],
    ['mcpnet.Base64Decode(s string) []byte', ''],
    ['mcpnet.Xor(b, key []byte) []byte', ''],
    ['mcpnet.XXTEAEncrypt(b, key []byte) []byte', ''],
    ['mcpnet.XXTEADecrypt(b, key []byte) []byte', ''],
    ['mcpnet.Log(s string)', '// 输出日志'],
    ['', ''],
    ['// ---- 示例：按你的條件建立窗口 ----', ''],
    ['func OnWebSocket() {', ''],
    ['    if mcpnet.Direction() == "in" && shouldOpenTool(mcpnet.Body()) {', ''],
    ['        html := `<button onclick="parent.postMessage({type:\'mcpnet-plugin-action\',action:\'refresh\'},\'*\')">Refresh</button>`', ''],
    ['        mcpnet.OpenWindow("My plugin tool", html)', ''],
    ['    }', ''],
    ['}', '']
  ].map(function (p) { return p[0] + (p[1] ? '  ' + p[1] : ''); }).join('\n');

  var SN_THEMES = [{"key":"shine","name":"OG Shine 墨绿","surface":"#0a0a0a","accent":"#50ecb3","text":"#ebebeb","dark":true,"bgKind":"solid","bgData":[]},{"key":"dark","name":"深灰","surface":"#1c1d1c","accent":"#b5cea5","text":"#dedfdd","dark":true,"bgKind":"solid","bgData":[]},{"key":"alpine","name":"高山","surface":"#152423","accent":"#a8d7ac","text":"#e2ede4","dark":true,"bgKind":"linear","bgData":["#0d1a19","#1d3230"]},{"key":"tide","name":"潮汐","surface":"#0e1d27","accent":"#73d4cd","text":"#dbe9ec","dark":true,"bgKind":"radial","bgData":["#123043","#0a1620"]},{"key":"orbit","name":"轨道","surface":"#171624","accent":"#c2aff7","text":"#e9e5f7","dark":true,"bgKind":"mesh","bgData":["#2b2f7a","#5a2f7e","#0f5368"]},{"key":"ember","name":"余烬","surface":"#1d1310","accent":"#f0a066","text":"#f3e5da","dark":true,"bgKind":"radial","bgData":["#4a2313","#180d09"]},{"key":"aurora","name":"极光","surface":"#14141f","accent":"#8de0b5","text":"#e3e2ee","dark":true,"bgKind":"mesh","bgData":["#1b3a4b","#2d4a3e","#3d2a5c"]},{"key":"phosphor","name":"磷光","surface":"#101c18","accent":"#a8e7ac","text":"#d5efdc","dark":true,"bgKind":"linear","bgData":["#0a1613","#16302a"]},{"key":"nebula","name":"星云","surface":"#0f141d","accent":"#f3a37a","text":"#e3e8f0","dark":true,"bgKind":"mesh","bgData":["#3a1f4d","#1a2a52","#4d2a1f"]},{"key":"midnight","name":"午夜","surface":"#11121d","accent":"#a5b4ff","text":"#e6e8f5","dark":true,"bgKind":"linear","bgData":["#0c0d17","#1a1c33"]},{"key":"dusk","name":"黄昏","surface":"#171220","accent":"#ffb38a","text":"#f3e7e1","dark":true,"bgKind":"radial","bgData":["#e2725b","#1d1430"]},{"key":"graphite","name":"石墨","surface":"#14161c","accent":"#c9d1e0","text":"#e4e7ee","dark":true,"bgKind":"solid","bgData":[]},{"key":"carbon","name":"碳黑","surface":"#191a1f","accent":"#9daecc","text":"#e1e3e9","dark":true,"bgKind":"solid","bgData":[]},{"key":"forest","name":"深林","surface":"#0f1a14","accent":"#7fc98a","text":"#dcefe0","dark":true,"bgKind":"linear","bgData":["#0a120d","#16281c"]},{"key":"ocean","name":"深海","surface":"#0a1620","accent":"#5fb3d4","text":"#d8e9f2","dark":true,"bgKind":"radial","bgData":["#123a52","#071018"]},{"key":"light","name":"浅色","surface":"#f0f2ef","accent":"#b5cea5","text":"#232b26","dark":false,"bgKind":"solid","bgData":[]},{"key":"dune","name":"沙丘","surface":"#f4eee5","accent":"#ac613e","text":"#382b23","dark":false,"bgKind":"linear","bgData":["#fbf6ee","#e8dcc8"]},{"key":"blossom","name":"花开","surface":"#f6eeee","accent":"#b4566f","text":"#3d292f","dark":false,"bgKind":"radial","bgData":["#fdf3f5","#f0dfe4"]},{"key":"after-rain","name":"雨后","surface":"#e9efeb","accent":"#557b71","text":"#1c312c","dark":false,"bgKind":"linear","bgData":["#f2f7f4","#dde8e2"]},{"key":"kanagawa","name":"神奈川","surface":"#f2ebd9","accent":"#2f5d8c","text":"#1f2a38","dark":false,"bgKind":"mesh","bgData":["#f5ecd6","#d6e3f0","#e8dfc8"]},{"key":"violet-hour","name":"紫罗兰时刻","surface":"#eceaf5","accent":"#7560ae","text":"#2e2842","dark":false,"bgKind":"mesh","bgData":["#dccdfb","#c6d5fb","#f5d6ee"]},{"key":"mirage","name":"海市蜃楼","surface":"#f3ebdf","accent":"#b0603b","text":"#372921","dark":false,"bgKind":"linear","bgData":["#faf4ea","#e6d5be"]},{"key":"frost","name":"霜白","surface":"#eaf0f6","accent":"#3c72ad","text":"#1c2a39","dark":false,"bgKind":"radial","bgData":["#f7fafd","#d8e5f0"]},{"key":"peach","name":"蜜桃","surface":"#fbf2ee","accent":"#d2604a","text":"#3a2621","dark":false,"bgKind":"mesh","bgData":["#ffd2c2","#ffc4d6","#fde6c4"]},{"key":"lilac","name":"丁香","surface":"#f4f1fb","accent":"#6c55c4","text":"#2b2440","dark":false,"bgKind":"mesh","bgData":["#dccdfb","#c6d5fb","#f5d6ee"]},{"key":"seafoam","name":"海沫","surface":"#eef6f2","accent":"#2a8a6e","text":"#1b362e","dark":false,"bgKind":"linear","bgData":["#c6efe0","#d6ecf6"]},{"key":"daybreak","name":"破晓","surface":"#f2f4f9","accent":"#3f6fc4","text":"#1f2a40","dark":false,"bgKind":"linear","bgData":["#ffffff","#dde5f2"]},{"key":"sherbet","name":"雪酪","surface":"#f8f2f4","accent":"#d1557b","text":"#38222c","dark":false,"bgKind":"radial","bgData":["#fff6f8","#f2dfe6"]},{"key":"paper","name":"纸张","surface":"#f1ede3","accent":"#8c6950","text":"#312d26","dark":false,"bgKind":"solid","bgData":[]},{"key":"sky","name":"晴空","surface":"#eef4fa","accent":"#3d8bd4","text":"#1c2c3d","dark":false,"bgKind":"linear","bgData":["#ffffff","#cfe4f7"]},{"key":"matcha","name":"抹茶","surface":"#f0f5ec","accent":"#5d8a4a","text":"#25301f","dark":false,"bgKind":"linear","bgData":["#f7fbf2","#ddebd2"]},{"key":"dark_blue_gray","name":"深色灰蓝","surface":"#1c1f26","accent":"#4a86e8","text":"#e6e9ef","dark":true,"bgKind":"solid","bgData":[]},{"key":"light_neutral","name":"浅色中性","surface":"#f4f5f7","accent":"#2d6fce","text":"#20242b","dark":false,"bgKind":"solid","bgData":[]},{"key":"graphite_contrast","name":"石墨高对比","surface":"#181818","accent":"#54a8ff","text":"#ffffff","dark":true,"bgKind":"solid","bgData":[]}];

  SN_THEMES = SN_THEMES.concat([
    {key:'coral-reef',name:'珊瑚礁',surface:'#fff3ed',accent:'#b9432c',text:'#38231e',dark:false,bgKind:'mesh',bgData:['#ffd0bb','#ffb7a0','#f5dfce']},
    {key:'lavender-mist',name:'薰衣草霧',surface:'#f4f1ff',accent:'#6047a8',text:'#2a243d',dark:false,bgKind:'mesh',bgData:['#ded2ff','#c8dcff','#f5d7f0']},
    {key:'glacier',name:'冰川',surface:'#edf7fb',accent:'#216783',text:'#1b3039',dark:false,bgKind:'linear',bgData:['#d8f1f7','#cadcf2']},
    {key:'citrus-grove',name:'柑橘園',surface:'#f5f8e9',accent:'#526f2d',text:'#29321f',dark:false,bgKind:'radial',bgData:['#e5f2b9','#d8e9cd']},
    {key:'rose-quartz',name:'玫瑰石英',surface:'#fff0f4',accent:'#a23d59',text:'#39232c',dark:false,bgKind:'mesh',bgData:['#ffd4df','#f5c5e0','#ffe4cf']},
    {key:'blue-hour-light',name:'藍調時刻',surface:'#eef2ff',accent:'#354f9b',text:'#222b45',dark:false,bgKind:'linear',bgData:['#d9e2ff','#d7eafa']},
    {key:'honey-oat',name:'蜂蜜燕麥',surface:'#fff8e9',accent:'#80531e',text:'#352a1d',dark:false,bgKind:'radial',bgData:['#ffe8b8','#f1d8b5']},
    {key:'mint-frost',name:'薄荷霜',surface:'#eaf8f1',accent:'#17684e',text:'#1b342b',dark:false,bgKind:'linear',bgData:['#c8f0df','#d1eaf1']},
    {key:'volcanic-glass',name:'火山玻璃',surface:'#1b1518',accent:'#ff9a72',text:'#f5e8e3',dark:true,bgKind:'mesh',bgData:['#572c32','#38243e','#63351f']},
    {key:'deep-indigo',name:'深靛藍',surface:'#11172a',accent:'#aab8ff',text:'#e8ecff',dark:true,bgKind:'linear',bgData:['#26396e','#332b68']},
    {key:'emerald-night',name:'翡翠之夜',surface:'#0e1e19',accent:'#73d9a6',text:'#e1f2e8',dark:true,bgKind:'radial',bgData:['#1e5943','#102c27']},
    {key:'arctic-night',name:'極地夜色',surface:'#101e29',accent:'#70d5ec',text:'#e0f0f5',dark:true,bgKind:'linear',bgData:['#1c4d67','#17354d']},
    {key:'plum-eclipse',name:'梅子蝕',surface:'#201528',accent:'#dda4f3',text:'#f1e6f6',dark:true,bgKind:'mesh',bgData:['#54336f','#352d69','#633355']},
    {key:'oxblood',name:'赤褐酒紅',surface:'#241516',accent:'#ff9b91',text:'#f5e7e5',dark:true,bgKind:'radial',bgData:['#663139','#3d202d']},
    {key:'dark-teal',name:'深青之境',surface:'#102321',accent:'#6ee7cf',text:'#e0f3ef',dark:true,bgKind:'linear',bgData:['#1e544e','#173d42']},
    {key:'midnight-gold',name:'午夜鎏金',surface:'#211d14',accent:'#f1ca79',text:'#f4edda',dark:true,bgKind:'mesh',bgData:['#59452a','#3e3524','#5b3926']}
  ]);

  function themeCard(t) {
    var card = el('div');
    card.style.cssText = 'position:relative;width:150px;height:104px;border-radius:8px;cursor:pointer;overflow:hidden;border:2px solid transparent;box-sizing:border-box;flex:0 0 auto;';
    var bg = card;
    if (t.bgKind === 'linear' && t.bgData.length >= 2) {
      bg.style.background = 'linear-gradient(132deg,color-mix(in srgb,' + t.bgData[0] + ' 68%,' + t.surface + ') 0%, ' + t.surface + ' 48%, color-mix(in srgb,' + t.bgData[t.bgData.length - 1] + ' 60%,' + t.surface + ') 100%)';
    } else if (t.bgKind === 'radial' && t.bgData.length >= 2) {
      bg.style.background = 'radial-gradient(ellipse at 12% 4%,color-mix(in srgb,' + t.bgData[0] + ' 68%,' + t.surface + ') 0%,transparent 58%),linear-gradient(150deg,' + t.surface + ',color-mix(in srgb,' + t.bgData[t.bgData.length - 1] + ' 54%,' + t.surface + '))';
    } else if (t.bgKind === 'mesh' && t.bgData.length >= 3) {
      bg.style.background = 'radial-gradient(ellipse at 8% 4%,color-mix(in srgb,' + t.bgData[0] + ' 66%,' + t.surface + ') 0%,transparent 50%),radial-gradient(ellipse at 92% 12%,color-mix(in srgb,' + t.bgData[1] + ' 58%,' + t.surface + ') 0%,transparent 48%),radial-gradient(ellipse at 48% 100%,color-mix(in srgb,' + t.bgData[2] + ' 52%,' + t.surface + ') 0%,transparent 55%),' + t.surface;
    } else {
      bg.style.background = t.surface;
    }
    var inner = el('div');
    inner.style.cssText = 'position:absolute;inset:0;padding:9px 10px;display:flex;flex-direction:column;';
    var bar = el('div');
    bar.style.cssText = 'display:flex;align-items:center;gap:5px;';
    var dot = el('span');
    dot.style.cssText = 'width:8px;height:8px;border-radius:50%;background:' + t.accent + ';flex:0 0 auto;';
    var nm = el('span');
    nm.style.cssText = 'font-size:11px;color:' + t.text + ';opacity:.92;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
    nm.textContent = t.name;
    bar.appendChild(dot); bar.appendChild(nm);
    inner.appendChild(bar);
    var card1 = el('div');
    card1.style.cssText = 'margin-top:8px;height:22px;border-radius:4px;background:' + (t.dark ? 'rgba(255,255,255,.10)' : 'rgba(30,45,60,.055)') + ';border:1px solid ' + (t.dark ? 'rgba(255,255,255,.14)' : 'rgba(30,45,60,.09)') + ';';
    inner.appendChild(card1);
    var row = el('div');
    row.style.cssText = 'margin-top:6px;display:flex;gap:5px;';
    var b1 = el('div');
    b1.style.cssText = 'flex:1 1 auto;height:16px;border-radius:3px;background:' + t.accent + ';opacity:.85;';
    var b2 = el('div');
    b2.style.cssText = 'width:34px;height:16px;border-radius:3px;background:' + (t.dark ? 'rgba(255,255,255,.18)' : 'rgba(30,45,60,.09)') + ';';
    row.appendChild(b1); row.appendChild(b2);
    inner.appendChild(row);
    card.appendChild(inner);
    card.dataset.key = t.key;
    card.dataset.text = t.text;
    card.dataset.surface = t.surface;
    return card;
  }

  function applyTheme(t) {
    if (window.ApplyAppPalette) window.ApplyAppPalette(t);
    if (window.ApplyAppTheme) window.ApplyAppTheme(!!t.dark);
    else if (window.Theme) window.Theme.IsDark = !!t.dark;
  }

  function buildThemeItem() {
    var panel = el('div');
    panel.id = 'sn-theme-panel';
    panel.style.cssText = 'padding:4px 0 2px;';
    var hint = el('div', '', L('选择一个界面配色，立即生效并自动保存。',
      '選擇一個介面配色，立即生效並自動保存。',
      'Pick a colour scheme. It applies immediately and is saved.'));
    hint.style.cssText = 'color:#aab7c2;font-size:12px;margin:2px 0 10px;';
    panel.appendChild(hint);

    var filterBar = el('div');
    filterBar.style.cssText = 'display:flex;flex-wrap:wrap;gap:6px;margin:0 0 10px;';
    panel.appendChild(filterBar);

    var grid = el('div');
    grid.style.cssText = 'display:flex;flex-wrap:wrap;gap:10px;';
    panel.appendChild(grid);

    var status = el('div');
    status.style.cssText = 'font-size:12px;color:#67c23a;margin-top:10px;min-height:16px;';
    panel.appendChild(status);

    var cards = [];
    var activeFilter = 'all';
    var filterDefs = [
      ['all', L('全部', '全部', 'All'), function () { return true; }],
      ['light', L('淺色', '淺色', 'Light'), function (t) { return !t.dark; }],
      ['dark', L('深色', '深色', 'Dark'), function (t) { return !!t.dark; }],
      ['gradient', L('漸層', '漸層', 'Gradients'), function (t) { return t.bgKind !== 'solid'; }]
    ];
    var filterButtons = [];
    function updateFilterButtons() {
      filterButtons.forEach(function (item) {
        var active = item.key === activeFilter;
        item.button.style.borderColor = active ? 'var(--app-accent,' + (activeFilter === 'dark' ? '#70d5ec' : '#2a8a6e') + ')' : 'var(--app-border,#66717a)';
        item.button.style.color = active ? 'var(--app-text,#e8edf2)' : 'var(--app-muted-text,#aab7c2)';
        item.button.style.background = active ? 'var(--app-row-selected,rgba(80,160,140,.16))' : 'var(--app-surface-raised,transparent)';
      });
      cards.forEach(function (item) { item.card.style.display = filterDefs.find(function (d) { return d[0] === activeFilter; })[2](item.theme) ? '' : 'none'; });
    }
    filterDefs.forEach(function (definition) {
      var button = el('button', '', definition[1] + ' · ' + SN_THEMES.filter(definition[2]).length);
      button.type = 'button';
      button.style.cssText = 'border:1px solid var(--app-border,#66717a);border-radius:14px;padding:4px 10px;font-size:12px;cursor:pointer;transition:background .15s,border-color .15s,color .15s;';
      button.addEventListener('click', function () { activeFilter = definition[0]; updateFilterButtons(); });
      filterBar.appendChild(button);
      filterButtons.push({key:definition[0],button:button});
    });
    SN_THEMES.forEach(function (t) {
      var c = themeCard(t);
      c.addEventListener('click', function () {
        cards.forEach(function (x) { x.card.style.borderColor = 'transparent'; });
        c.style.borderColor = t.accent;
        applyTheme(t);
        doCommand('保存主题', { Key: t.key }).then(function () {
          status.style.color = '#67c23a';
          status.textContent = L('已应用：', '已套用：', 'Applied: ') + t.name;
        }).catch(function (e) {
          status.style.color = '#f56c6c';
          status.textContent = String(e && e.message ? e.message : e);
        });
      });
      grid.appendChild(c);
      cards.push({card:c,theme:t});
    });
    updateFilterButtons();

    doCommand('主题信息', null).then(function (info) {
      var key = info && info.key ? info.key : 'shine';
      var found = null;
      SN_THEMES.forEach(function (t) { if (t.key === key) found = t; });
      if (!found) found = SN_THEMES[0];
      cards.forEach(function (x) { if (x.card.dataset.key === found.key) x.card.style.borderColor = found.accent; });
      applyTheme(found);
    }).catch(function () {});

    return collapseItem('M12 3a9 9 0 1 0 0 18 4 4 0 0 0 0-8 4 4 0 0 1 0-8', L('界面主题', '介面主題', 'Theme'), null, panel, false);
  }

  function buildPluginFeaturesItem() {
    var panel = el('div');
    panel.id = 'sn-plugin-panel';
    panel.style.cssText = 'padding:4px 0 2px;';

    var hint = el('div', '', L('插件保存在本機。用 Go hooks 自由判斷協議、URL、字串、正則或位元組格式並處理流量；插件程式可呼叫 mcpnet.OpenWindow(title, html) 建立內置工具窗口，或用 mcpnet.OpenExternalURL(url) 開啟外部頁面，不需要客戶端為每個插件新增專用按鈕。Go 程式碼在本機執行，請只啟用可信插件。',
      '插件保存在本機。用 Go hooks 自由判斷協議、URL、字串、正則或位元組格式並處理流量；插件程式可呼叫 mcpnet.OpenWindow(title, html) 建立內置工具窗口，或用 mcpnet.OpenExternalURL(url) 開啟外部頁面，不需要客戶端為每個插件新增專用按鈕。Go 程式碼在本機執行，請只啟用可信插件。',
      'Plugins live locally. Go hooks can inspect protocols, URLs, strings, regexps, or byte patterns and process traffic. Plugin code can call mcpnet.OpenWindow(title, html) for an in-app tool window or mcpnet.OpenExternalURL(url) for an external page; the client does not add plugin-specific buttons. Go code runs locally; enable trusted plugins only.'));
    hint.style.cssText = 'color:var(--app-muted-text,#aab7c2);font-size:12px;line-height:1.6;margin:2px 0 8px;';
    panel.appendChild(hint);

    var listBox = el('div');
    listBox.style.cssText = 'border:1px solid var(--app-border,#3c3c3c);border-radius:4px;background:var(--app-surface-raised,#1f1f1f);max-height:150px;overflow:auto;color:var(--app-text,#d4d4d4);';
    panel.appendChild(listBox);

    var bar = el('div');
    bar.style.cssText = 'display:flex;align-items:center;flex-wrap:wrap;gap:8px;margin:10px 0 4px;';
    var newBtn = el('button', 'el-button el-button--small el-button--primary');
    newBtn.textContent = L('新建插件', '新建插件', 'New plugin');
    var importBtn = el('button', 'el-button el-button--small');
    importBtn.textContent = L('導入插件', '導入插件', 'Import plugin');
    importBtn.setAttribute('aria-haspopup', 'dialog');
    var refreshBtn = el('button', 'el-button el-button--small');
    refreshBtn.textContent = L('刷新列表', '刷新清單', 'Refresh');
    refreshBtn.title = L('重新读取本地插件状态', '重新讀取本機插件狀態', 'Reload local plugin status');
    var dirText = el('span');
    dirText.style.cssText = 'color:var(--app-muted-text,#7f8c98);font-size:11px;flex:1 1 auto;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
    var status = el('span');
    status.style.cssText = 'font-size:12px;color:#aab7c2;';
    bar.appendChild(newBtn); bar.appendChild(importBtn); bar.appendChild(refreshBtn); bar.appendChild(dirText); bar.appendChild(status);
    panel.appendChild(bar);

    // ---- community plugin import chooser ----
    var importOverlay = el('div');
    importOverlay.setAttribute('role', 'presentation');
    importOverlay.style.cssText = 'display:none;position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:99998;align-items:center;justify-content:center;padding:16px;box-sizing:border-box;';
    var importModal = el('div');
    importModal.setAttribute('role', 'dialog');
    importModal.setAttribute('aria-modal', 'true');
    importModal.setAttribute('aria-labelledby', 'sn-plugin-import-title');
    importModal.style.cssText = 'width:min(480px,96vw);background:var(--app-surface,#1b1b1b);color:var(--app-text,#d4d4d4);border:1px solid var(--app-border,#3c3c3c);border-radius:6px;box-shadow:0 12px 40px rgba(0,0,0,.5);overflow:hidden;';
    var importTitle = el('div', '', L('導入社群插件', '導入社群插件', 'Import a community plugin'));
    importTitle.id = 'sn-plugin-import-title';
    importTitle.style.cssText = 'padding:12px 14px;background:var(--app-header,#202020);border-bottom:1px solid var(--app-border,#333);font-size:14px;font-weight:600;';
    var importHint = el('div', '', L(
      '選擇插件 ZIP 或資料夾。格式需包含 main.go 和一個 JSON 清單（建議命名 plugin.json）。導入後預設停用。',
      '選擇插件 ZIP 或資料夾。格式需包含 main.go 和一個 JSON 清單（建議命名 plugin.json）。導入後預設停用。',
      'Choose a plugin ZIP or folder containing main.go and one JSON manifest (plugin.json recommended). Imported plugins are disabled by default.'
    ));
    importHint.style.cssText = 'padding:12px 14px 4px;font-size:12px;line-height:1.6;color:var(--app-muted-text,#aab7c2);';
    var importWarning = el('div', '', L(
      '安全提醒：插件程式會在本機執行，啟用後可能讀取或修改流量。只導入你信任的來源。',
      '安全提醒：插件程式會在本機執行，啟用後可能讀取或修改流量。只導入你信任的來源。',
      'Security: plugin code runs locally and may read or modify traffic when enabled. Import code only from sources you trust.'
    ));
    importWarning.style.cssText = 'margin:8px 14px;padding:9px 10px;border:1px solid var(--app-border,#5b5140);border-radius:4px;color:var(--app-text,#d4d4d4);background:rgba(230,162,60,.10);font-size:12px;line-height:1.5;';
    var importActions = el('div');
    importActions.style.cssText = 'display:flex;flex-wrap:wrap;gap:8px;padding:8px 14px 14px;';
    var importStatus = el('div');
    importStatus.setAttribute('role', 'status');
    importStatus.setAttribute('aria-live', 'polite');
    importStatus.style.cssText = 'padding:0 14px 8px;font-size:12px;overflow-wrap:anywhere;';
    var importZipBtn = el('button', 'el-button el-button--small el-button--primary');
    importZipBtn.textContent = L('選擇 ZIP', '選擇 ZIP', 'Choose ZIP');
    var importFolderBtn = el('button', 'el-button el-button--small');
    importFolderBtn.textContent = L('選擇資料夾', '選擇資料夾', 'Choose folder');
    var importCancelBtn = el('button', 'el-button el-button--small');
    importCancelBtn.textContent = L('取消', '取消', 'Cancel');
    [importZipBtn, importFolderBtn, importCancelBtn].forEach(function (button) {
      button.style.minHeight = '36px';
      importActions.appendChild(button);
    });
    importModal.appendChild(importTitle);
    importModal.appendChild(importHint);
    importModal.appendChild(importWarning);
    importModal.appendChild(importStatus);
    importModal.appendChild(importActions);
    importOverlay.appendChild(importModal);
    panel.appendChild(importOverlay);

    // Use WebView file inputs for package selection. This avoids opening the
    // native Wails dialog from a bound method, which could terminate the app.
    var zipInput = el('input');
    zipInput.type = 'file';
    zipInput.accept = '.zip,application/zip';
    zipInput.style.cssText = 'display:none;';
    var folderInput = el('input');
    folderInput.type = 'file';
    folderInput.setAttribute('webkitdirectory', '');
    folderInput.setAttribute('directory', '');
    folderInput.style.cssText = 'display:none;';
    panel.appendChild(zipInput);
    panel.appendChild(folderInput);

    function setImportStatus(message, isError) {
      importStatus.style.color = isError ? '#f56c6c' : 'var(--app-muted-text,#aab7c2)';
      importStatus.textContent = message || '';
    }
    function setImportBusy(busy) {
      importZipBtn.disabled = busy;
      importFolderBtn.disabled = busy;
      importCancelBtn.disabled = busy;
    }
    function submitImport(command, args) {
      setImportBusy(true);
      setImportStatus(L('正在驗證並導入插件…', '正在驗證並匯入插件…', 'Validating and importing plugin…'), false);
      return doCommand(command, args).then(function (info) {
        if (!info || info.error) throw new Error(info && info.error ? info.error : '插件匯入失敗');
        state.plugins = info;
        render(L('插件已導入並保持停用；確認來源可信後再勾選啟用。', '插件已匯入並保持停用；確認來源可信後再勾選啟用。', 'Plugin imported disabled. Verify its source before enabling it.'));
        setImportStatus(L('導入完成；插件預設停用。', '匯入完成；插件預設停用。', 'Import complete; the plugin is disabled by default.'), false);
      }).catch(function (error) {
        setImportStatus(String(error && error.message ? error.message : error), true);
      }).finally(function () {
        setImportBusy(false);
      });
    }
    function readText(file) {
      return new Promise(function (resolve, reject) {
        var reader = new FileReader();
        reader.onload = function () { resolve(String(reader.result || '')); };
        reader.onerror = function () { reject(new Error('讀取插件檔案失敗')); };
        reader.readAsText(file, 'UTF-8');
      });
    }
    function fileAsBase64(file) {
      return new Promise(function (resolve, reject) {
        var reader = new FileReader();
        reader.onload = function () {
          var value = String(reader.result || '');
          var comma = value.indexOf(',');
          if (comma < 0) reject(new Error('ZIP 檔案格式無效'));
          else resolve(value.slice(comma + 1));
        };
        reader.onerror = function () { reject(new Error('讀取 ZIP 失敗')); };
        reader.readAsDataURL(file);
      });
    }
    function handlePickerCancel() {
      setImportBusy(false);
      setImportStatus(L('已取消檔案選擇，程式仍在運行。', '已取消檔案選擇，程式仍在執行。', 'Selection cancelled; the application is still running.'), false);
    }
    zipInput.addEventListener('cancel', handlePickerCancel);
    folderInput.addEventListener('cancel', handlePickerCancel);
    zipInput.addEventListener('change', function () {
      var file = zipInput.files && zipInput.files[0];
      zipInput.value = '';
      if (!file) { handlePickerCancel(); return; }
      if (file.size > 16 * 1024 * 1024) { setImportStatus(L('ZIP 檔案不可超過 16 MB', 'ZIP 檔案不可超過 16 MB', 'ZIP file must be 16 MB or smaller.'), true); return; }
      setImportBusy(true);
      fileAsBase64(file).then(function (Data) {
        return submitImport('導入插件ZIP', { Data: Data });
      }).catch(function (error) {
        setImportStatus(String(error && error.message ? error.message : error), true);
      }).finally(function () { setImportBusy(false); });
    });
    folderInput.addEventListener('change', function () {
      var files = Array.from(folderInput.files || []);
      folderInput.value = '';
      if (!files.length) { handlePickerCancel(); return; }
      var mainFiles = [];
      var jsonFiles = [];
      files.forEach(function (file) {
        var parts = String(file.webkitRelativePath || file.name).replace(/\\/g, '/').split('/');
        if (parts.length !== 2) return;
        if (parts[1] === 'main.go') mainFiles.push(file);
        else if (/\.json$/i.test(parts[1])) jsonFiles.push(file);
      });
      if (mainFiles.length !== 1 || jsonFiles.length !== 1) {
        setImportStatus(L('所選資料夾根目錄需有一個 main.go 和一個 JSON 清單。', '所選資料夾根目錄需有一個 main.go 和一個 JSON 清單。', 'The selected folder must contain exactly one main.go and one JSON manifest at its root.'), true);
        return;
      }
      if (mainFiles[0].size > 4 * 1024 * 1024 || jsonFiles[0].size > 4 * 1024 * 1024) {
        setImportStatus(L('插件單一檔案不可超過 4 MB。', '插件單一檔案不可超過 4 MB。', 'Each plugin file must be 4 MB or smaller.'), true);
        return;
      }
      setImportBusy(true);
      Promise.all([readText(jsonFiles[0]), readText(mainFiles[0])]).then(function (values) {
        return submitImport('導入插件資料夾', { Manifest: values[0], Source: values[1] });
      }).catch(function (error) {
        setImportStatus(String(error && error.message ? error.message : error), true);
      }).finally(function () { setImportBusy(false); });
    });

    function closeImportDialog() {
      importOverlay.style.display = 'none';
      importBtn.focus();
    }
    function beginPluginImport(type) {
      setImportStatus(L('選擇取消不會關閉程式；請選取 ZIP 或插件資料夾。', '選擇取消不會關閉程式；請選取 ZIP 或插件資料夾。', 'Cancelling selection will not close the app. Choose a ZIP or plugin folder.'), false);
      var input = type === 'zip' ? zipInput : folderInput;
      input.value = '';
      input.click();
    }
    importBtn.addEventListener('click', function () {
      importOverlay.style.display = 'flex';
      importZipBtn.focus();
    });
    importZipBtn.addEventListener('click', function () { beginPluginImport('zip'); });
    importFolderBtn.addEventListener('click', function () { beginPluginImport('folder'); });
    importCancelBtn.addEventListener('click', closeImportDialog);
    importOverlay.addEventListener('click', function (event) { if (event.target === importOverlay) closeImportDialog(); });
    importModal.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') { event.preventDefault(); closeImportDialog(); }
    });

    // ---- full editor overlay ----
    var overlay = el('div');
    overlay.style.cssText = 'display:none;position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:99999;align-items:center;justify-content:center;';
    var modal = el('div');
    modal.style.cssText = 'width:min(1440px,96vw);height:min(900px,92vh);background:var(--app-surface,#1b1b1b);color:var(--app-text,#d4d4d4);border:1px solid var(--app-border,#3c3c3c);border-radius:6px;display:flex;flex-direction:column;overflow:hidden;box-shadow:0 12px 40px rgba(0,0,0,.5);';
    var mHead = el('div');
    mHead.style.cssText = 'display:flex;align-items:center;gap:10px;padding:10px 14px;border-bottom:1px solid var(--app-border,#333);background:var(--app-header,#202020);';
    var mTitle = el('div');
    mTitle.style.cssText = 'color:var(--app-text,#e6edf3);font-size:13px;flex:1 1 auto;';
    mTitle.textContent = L('插件编辑器', '插件編輯器', 'Plugin editor');
    var mClose = el('button', 'el-button el-button--small');
    mClose.textContent = '✕';
    mHead.appendChild(mTitle); mHead.appendChild(mClose);
    modal.appendChild(mHead);

    var meta = el('div');
    meta.style.cssText = 'display:grid;grid-template-columns:repeat(3,1fr);gap:6px;padding:10px 14px 6px;';
    function field(ph, val) {
      var i = el('input', 'el-input__inner');
      i.placeholder = ph;
      if (val) i.value = val;
      i.style.cssText = 'width:100%;box-sizing:border-box;background:var(--app-surface-raised,#1f1f1f);color:var(--app-text,#d4d4d4);border:1px solid var(--app-border,#3c3c3c);border-radius:4px;padding:5px 7px;font-size:12px;';
      meta.appendChild(i);
      return i;
    }
    var fID = field('id（留空自动生成）');
    var fName = field('名称 *');
    var fVersion = field('版本 1.0.0');
    var fAuthor = field('作者');
    var fOrder = field('order 100');
    var fDesc = field('描述');
    modal.appendChild(meta);

    var body = el('div');
    body.style.cssText = 'flex:1 1 auto;display:flex;min-height:0;padding:0 14px 10px;gap:10px;';
    var codeWrap = el('div');
    codeWrap.style.cssText = 'flex:1 1 72%;display:flex;min-width:0;min-height:0;border:1px solid var(--app-border,#3c3c3c);border-radius:4px;overflow:hidden;background:var(--app-surface,#141414);';
    var editorHost = el('div');
    editorHost.style.cssText = 'flex:1 1 auto;width:100%;height:100%;min-width:0;min-height:0;text-align:left;';
    var fallbackSrc = el('textarea');
    fallbackSrc.spellcheck = false;
    fallbackSrc.style.cssText = 'width:100%;height:100%;box-sizing:border-box;background:var(--app-surface,#141414);color:var(--app-text,#d4d4d4);border:0;outline:0;padding:12px;font:14px/1.6 Consolas,monospace;resize:none;white-space:pre;overflow:auto;tab-size:4;';
    var codeEditor = null;
    codeWrap.appendChild(editorHost);
    body.appendChild(codeWrap);

    var docs = el('div');
    docs.style.cssText = 'flex:0 1 28%;min-width:220px;border:1px solid var(--app-border,#3c3c3c);border-radius:4px;background:var(--app-surface-raised,#161616);color:var(--app-muted-text,#9fb0be);font:12px/1.7 Consolas,monospace;padding:10px 12px;overflow:auto;white-space:pre-wrap;';
    docs.textContent = PLUGIN_DOCS;
    body.appendChild(docs);
    modal.appendChild(body);

    var foot = el('div');
    foot.style.cssText = 'display:flex;align-items:center;gap:8px;padding:10px 14px;border-top:1px solid var(--app-border,#333);background:var(--app-header,#202020);';
    var saveBtn = el('button', 'el-button el-button--small el-button--primary');
    saveBtn.textContent = L('保存插件', '保存插件', 'Save plugin');
    var cancelBtn = el('button', 'el-button el-button--small');
    cancelBtn.textContent = L('取消', '取消', 'Cancel');
    var footMsg = el('span');
    footMsg.style.cssText = 'font-size:12px;color:var(--app-muted-text,#aab7c2);';
    foot.appendChild(saveBtn); foot.appendChild(cancelBtn); foot.appendChild(footMsg);
    modal.appendChild(foot);

    overlay.appendChild(modal);
    panel.appendChild(overlay);

    var editingId = '';
    function ensureCodeEditor() {
      if (codeEditor || fallbackSrc.parentNode) return;
      var monaco = window.MonacoEditor;
      if (monaco && monaco.editor) {
        codeEditor = monaco.editor.create(editorHost, {
          value: '', language: 'go',
          theme: window.Theme && window.Theme.IsDark ? 'vs-dark' : 'vs',
          automaticLayout: true, minimap: {enabled: false}, fontSize: 14,
          lineNumbers: 'on', tabSize: 4, insertSpaces: false,
          scrollBeyondLastLine: false, wordWrap: 'off',
          padding: {top: 10, bottom: 10},
          scrollbar: {verticalScrollbarSize: 12, horizontalScrollbarSize: 12}
        });
        window.addEventListener('darkThemeChange', function () {
          if (codeEditor) monaco.editor.setTheme(window.Theme && window.Theme.IsDark ? 'vs-dark' : 'vs');
        });
      } else {
        editorHost.appendChild(fallbackSrc);
      }
    }
    function setCode(value) {
      ensureCodeEditor();
      if (codeEditor) codeEditor.setValue(value);
      else fallbackSrc.value = value;
    }
    function getCode() {
      return codeEditor ? codeEditor.getValue() : fallbackSrc.value;
    }
    function openEditor(p, code) {
      editingId = p && p.id ? p.id : '';
      fID.value = editingId;
      fName.value = p && p.name ? p.name : '';
      fVersion.value = p && p.version ? p.version : '1.0.0';
      fAuthor.value = p && p.author ? p.author : '';
      fOrder.value = String((p && p.order) || 100);
      fDesc.value = p && p.description ? p.description : '';
      setCode(code || pluginTemplate());
      footMsg.textContent = '';
      overlay.style.display = 'flex';
      setTimeout(function () {
        if (codeEditor) { codeEditor.layout(); codeEditor.focus(); }
        else fallbackSrc.focus();
      }, 0);
    }
    function closeEditor() { overlay.style.display = 'none'; }

    mClose.addEventListener('click', closeEditor);
    cancelBtn.addEventListener('click', closeEditor);
    overlay.addEventListener('click', function (e) { if (e.target === overlay) closeEditor(); });

    saveBtn.addEventListener('click', function () {
      if (!fName.value.trim()) { footMsg.style.color = '#f56c6c'; footMsg.textContent = L('请填写名称', '請填寫名稱', 'Name required'); return; }
      var p = {
        id: editingId || fID.value.trim(),
        name: fName.value.trim(),
        version: fVersion.value.trim(),
        author: fAuthor.value.trim(),
        description: fDesc.value.trim(),
        enabled: true,
        order: parseInt(fOrder.value, 10) || 100
      };
      doCommand('保存插件', { Manifest: p, WithSource: 'true', Source: getCode() }).then(function (info) {
        if (info && info.error) { footMsg.style.color = '#f56c6c'; footMsg.textContent = info.error; return; }
        state.plugins = info || {};
        render();
        closeEditor();
        status.style.color = '#67c23a';
        status.textContent = L('已保存', '已保存', 'Saved');
      });
    });

    function render(messageOverride) {
      doCommand('插件列表', null).then(function (info) {
        state.plugins = info || {};
        dirText.textContent = info && info.dir ? info.dir : '';
        listBox.innerHTML = '';
        var items = (info && info.plugins) || [];
        status.style.color = messageOverride ? '#67c23a' : 'var(--app-muted-text,#8b98a5)';
        status.textContent = messageOverride || L('已读取 ' + items.length + ' 个本地插件', '已讀取 ' + items.length + ' 個本機插件', items.length + ' local plugin(s) loaded');
        if (!items.length) {
          var empty = el('div', '', L('还没有插件，点击「新建插件」开始。', '還沒有插件，點擊「新建插件」開始。', 'No plugins yet. Click "New plugin".'));
          empty.style.cssText = 'color:var(--app-muted-text,#8b98a5);font-size:12px;padding:10px;';
          listBox.appendChild(empty);
          return;
        }
        items.forEach(function (p) {
          var row = el('div');
          row.style.cssText = 'display:flex;flex-wrap:wrap;align-items:center;gap:8px;padding:7px 10px;border-bottom:1px solid var(--app-border,#333);';
          var cb = el('input');
          cb.type = 'checkbox';
          cb.checked = !!p.enabled;
          cb.setAttribute('aria-label', L('启用插件：' + (p.name || p.id), '啟用插件：' + (p.name || p.id), 'Enable plugin: ' + (p.name || p.id)));
          cb.title = cb.getAttribute('aria-label');
          cb.style.cssText = 'accent-color:var(--app-accent,#409eff);width:15px;height:15px;flex:0 0 auto;';
          cb.addEventListener('change', function () {
            var enabled = cb.checked;
            p.enabled = enabled;
            cb.disabled = true;
            saveMeta(p, enabled ? L('插件已启用', '插件已啟用', 'Plugin enabled') : L('插件已停用', '插件已停用', 'Plugin disabled')).then(function (saved) {
              if (!saved) {
                p.enabled = !enabled;
                cb.checked = !enabled;
              }
              cb.disabled = false;
            });
          });
          var main = el('div');
          main.style.cssText = 'flex:1 1 auto;min-width:0;';
          var t = el('div');
          t.style.cssText = 'color:var(--app-text,#dbe3ea);font-size:13px;';
          t.textContent = (p.name || p.id) + (p.version ? '  v' + p.version : '');
          var sub = el('div');
          sub.style.cssText = 'color:var(--app-muted-text,#8b98a5);font-size:11px;margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';
          sub.textContent = [p.author ? '@' + p.author : '', p.description || ''].filter(Boolean).join(' · ') || p.id;
          main.appendChild(t); main.appendChild(sub);
          var enabledState = el('span');
          enabledState.textContent = p.enabled ? L('已启用', '已啟用', 'Enabled') : L('已停用', '已停用', 'Disabled');
          enabledState.style.cssText = 'flex:0 0 auto;color:' + (p.enabled ? 'var(--app-accent,#409eff)' : 'var(--app-muted-text,#8b98a5)') + ';font-size:11px;';
          main.appendChild(enabledState);
          if (p.error) {
            var error = el('div');
            error.textContent = L('加载错误：', '載入錯誤：', 'Load error: ') + p.error;
            error.style.cssText = 'color:#f56c6c;font-size:11px;margin-top:3px;white-space:normal;overflow-wrap:anywhere;';
            main.appendChild(error);
          }
          var edit = el('button', 'el-button el-button--small');
          edit.textContent = L('编辑代码', '編輯程式碼', 'Edit');
          edit.addEventListener('click', function () {
            doCommand('插件源码', { Id: p.id }).then(function (code) {
              openEditor(p, code);
            });
          });
          var del = el('button', 'el-button el-button--small');
          del.textContent = L('删除', '刪除', 'Delete');
          del.addEventListener('click', function () {
            doCommand('删除插件', { Id: p.id }).then(function (info) {
              state.plugins = info || {}; render();
              status.style.color = '#67c23a';
              status.textContent = L('已删除', '已刪除', 'Deleted');
            });
          });
          row.appendChild(cb); row.appendChild(main); row.appendChild(edit); row.appendChild(del);
          listBox.appendChild(row);
        });
      }).catch(function (e) {
        status.style.color = '#f56c6c';
        status.textContent = String(e && e.message ? e.message : e);
      });
    }

    function saveMeta(p, okText) {
      return doCommand('设置插件状态', { Id: p.id || '', Enabled: !!p.enabled }).then(function (info) {
        if (info && info.error) { status.style.color = '#f56c6c'; status.textContent = info.error; return false; }
        state.plugins = info || {}; render();
        status.style.color = '#67c23a';
        status.textContent = okText || L('已保存', '已保存', 'Saved');
        return true;
      }).catch(function (e) {
        status.style.color = '#f56c6c';
        status.textContent = String(e && e.message ? e.message : e);
        return false;
      });
    }

    newBtn.addEventListener('click', function () { openEditor(null, pluginTemplate()); });
    refreshBtn.addEventListener('click', render);
    render();
    return collapseItem('M4 7h16M4 12h16M4 17h10M18 15l3 3-3 3', L('插件功能', '插件功能', 'Plugin Features'), null, panel, false);
  }

  function pluginTemplate() {
    return [
      'package main',
      '',
      'import "mcpnet"',
      '',
      '// Implement one or more hooks as needed.',
      '// OnHTTPRequest / OnHTTPResponse / OnWebSocket / OnUDP / OnURLDisplay',
      'func OnWebSocket() {',
      '\tif mcpnet.Direction() == "in" {',
      '\t\t// Decode with your own local rule, then use SetDisplayBody',
      '\t\t// when the decoded bytes should only appear in the UI.',
      '\t\t// mcpnet.SetDisplayBody(decoded)',
      '\t}',
      '}',
      ''
    ].join('\n');
  }
  function injectInner() {
    window.__snMcp.runs++; window.__snMcp.why = 'start';
    var dialogs = document.querySelectorAll('.ag-dialog');
    var dlg = null;
    for (var i = 0; i < dialogs.length; i++) {
      var title = dialogs[i].querySelector('.ag-panel-title-bar-title');
      var t = title ? (title.textContent || '') : '';
      if (/设置|設定|Settings/i.test(t)) { dlg = dialogs[i]; break; }
    }
    if (!dlg) { state.scrollDone = false; state.processScrolled = false; window.__snMcp.why = 'no-dlg'; return; }
    var main = dlg.querySelector('.el-main');
    if (!main) { window.__snMcp.why = 'no-main'; return; }
    maybeScrollProcess(dlg);
    var oldLang = main.querySelector('#sn-lang-box');
    if (oldLang) oldLang.remove();
    if (document.getElementById('sn-mcp-item') && document.getElementById('sn-lang-item') && document.getElementById('sn-codec-item') && document.getElementById('sn-theme-item')) return;
    window.__snMcp.why = 'creating';
    var langItem = buildLanguageItem();
    langItem.id = 'sn-lang-item';
    var mcpItem = buildMcpItem();
    mcpItem.id = 'sn-mcp-item';
    var codecItem = buildPluginFeaturesItem();
    codecItem.id = 'sn-codec-item';
    var themeItem = buildThemeItem();
    themeItem.id = 'sn-theme-item';
    var nativeCollapse = main.querySelector('.el-collapse');
    if (nativeCollapse) {
      nativeCollapse.insertBefore(mcpItem, nativeCollapse.firstChild);
      nativeCollapse.insertBefore(codecItem, nativeCollapse.firstChild);
      nativeCollapse.insertBefore(themeItem, nativeCollapse.firstChild);
      nativeCollapse.insertBefore(langItem, nativeCollapse.firstChild);
    } else {
      var wrap = el('div', 'el-collapse');
      wrap.appendChild(langItem);
      wrap.appendChild(themeItem);
      wrap.appendChild(codecItem);
      wrap.appendChild(mcpItem);
      main.insertBefore(wrap, main.firstChild);
    }
    var panel = mcpItem.querySelector('#' + MCP_ID);
    refresh(panel);
  }
  function inject() {
    try { injectInner(); } catch (e) { window.__snMcp.err = 'inject: ' + String(e); }
  }
  function hookHeader() {
    try {
      if (!window.vm || !window.vm.Header) return;
      ['ShowDrive', 'clickSettings'].forEach(function (name) {
        var f = window.vm.Header[name];
        if (typeof f !== 'function' || f.__snHook) return;
        var wrapped = function () {
          var r = f.apply(this, arguments);
          setTimeout(inject, 0);
          setTimeout(inject, 120);
          setTimeout(inject, 500);
          return r;
        };
        wrapped.__snHook = true;
        window.vm.Header[name] = wrapped;
      });
    } catch (e) {}
  }
  setInterval(hookHeader, 500);
  document.addEventListener('click', function () {
    setTimeout(inject, 0);
    setTimeout(inject, 150);
    setTimeout(inject, 600);
  }, true);

  var injectPending = false;
  function scheduleInject() {
    if (injectPending) return;
    injectPending = true;
    setTimeout(function () { injectPending = false; inject(); }, 50);
  }
  if (document.body) new MutationObserver(scheduleInject).observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['style', 'class'] });
  setInterval(inject, 1000);
  setInterval(function () {
    var panel = document.getElementById(MCP_ID);
    if (panel && panel.offsetParent !== null) refresh(panel);
  }, 3000);
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', inject);
  else inject();
})();
