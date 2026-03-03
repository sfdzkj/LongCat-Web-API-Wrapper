package admin

const loginHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width,initial-scale=1"/>
  <title>Admin Login</title>
  <style>
    body{font:14px/1.4 system-ui;margin:0;display:flex;align-items:center;justify-content:center;height:100vh;background:#121212;color:#e0e0e0}
    .card{background:#1e1e1e;padding:24px;border-radius:12px;box-shadow:0 4px 20px rgba(0,0,0,0.5);width:100%;max-width:360px}
    h3{margin:0 0 20px;text-align:center}
    input{width:100%;padding:12px;margin:10px 0;border-radius:6px;border:1px solid #333;background:#2c2c2c;color:#fff;box-sizing:border-box;font-size:14px}
    input:focus{outline:none;border-color:#3d5afe}
    button{width:100%;padding:12px;border-radius:6px;border:none;background:#3d5afe;color:#fff;cursor:pointer;font-weight:700;font-size:14px;transition:0.2s}
    button:hover{background:#536dfe}
    .msg{color:#ff5252;font-size:13px;margin-top:12px;text-align:center}
    .default-pwd{background:#263238;padding:12px;border-radius:6px;margin-bottom:20px;border:1px dashed #546e7a;text-align:center}
    .default-pwd b{color:#ffeb3b;font-family:monospace;font-size:18px;letter-spacing:1px}

  </style>
</head>
<body>
  <div class="card">
    <h3>LongCat Admin</h3>
    {{if .IsDefault}}
    <div class="default-pwd">
      <div style="color:#ff5252;font-weight:700;margin-bottom:6px">初始默认密码</div>
      <b>{{.DefaultPwd}}</b>
      <div style="font-size:11px;color:#90a4ae;margin-top:6px">登录后请立即修改密码</div>
    </div>
    {{end}}
    <input id="pwd" type="password" placeholder="请输入密码" onkeydown="if(event.key==='Enter')login()"/>
    <button onclick="login()">登录</button>
    <div id="err" class="msg"></div>
  </div>
  <script>
    async function login() {
      const err = document.getElementById('err');
      err.textContent = '';
      const password = document.getElementById('pwd').value;
      if (!password) { err.textContent = '请输入密码'; return; }
      const res = await fetch('/admin/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({password: password})
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({error: '登录失败'}));
        err.textContent = j.error || '登录失败';
        return;
      }
      location.href = '/admin';
    }
  </script>
</body>
</html>`

const dashHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width,initial-scale=1"/>
  <title>LongCat Admin</title>
  <style>
    :root {
      --bg: #050607;
      --bg-soft: #0b0d10;
      --sidebar-bg: #0a0c0f;
      --card-bg: #101318;
      --card-strong: #121720;
      --text: #e8edf5;
      --text-dim: #9aa7bc;
      --primary: #4ea1ff;
      --primary-hover: #77b8ff;
      --danger: #ff5f6d;
      --success: #4fd48b;
      --warning: #ffb454;
      --border: #1d2633;
      --input-bg: #0c1118;
      --sidebar-width: 252px;
    }
    *{box-sizing:border-box}
    body{font:14px/1.5 "IBM Plex Sans","Segoe UI",sans-serif;margin:0;background:radial-gradient(circle at 18% 8%, #17233a 0%, transparent 30%), radial-gradient(circle at 85% 92%, #142130 0%, transparent 25%), var(--bg);color:var(--text);display:flex;height:100vh;overflow:hidden}
    
    /* Sidebar */
    .sidebar { width: var(--sidebar-width); background: linear-gradient(180deg, rgba(255,255,255,0.02), transparent), var(--sidebar-bg); border-right: 1px solid var(--border); display: flex; flex-direction: column; flex-shrink: 0; }
    .sidebar-header { padding: 22px 24px; font-size: 19px; font-weight: 700; display: flex; align-items: center; gap: 10px; border-bottom: 1px solid var(--border); letter-spacing: .3px; }
    .sidebar-header span { color: var(--primary); }
    .nav { flex: 1; padding: 16px 0; overflow-y: auto; }
    .nav-item { padding: 12px 24px; cursor: pointer; display: flex; align-items: center; gap: 12px; color: var(--text-dim); transition: 0.2s; border-right: 3px solid transparent; }
    .nav-item:hover { background: rgba(255,255,255,0.04); color: var(--text); }
    .nav-item.active { background: linear-gradient(90deg, rgba(78,161,255,.18), rgba(78,161,255,.05)); color: #d2e8ff; border-right-color: var(--primary); }
    .sidebar-footer { padding: 16px; border-top: 1px solid var(--border); }
    
    /* Main Content */
    .main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
    .top-bar { height: 66px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; padding: 0 24px; background: rgba(10,12,15,0.85); backdrop-filter: blur(6px); }
    .content-area { flex: 1; overflow-y: auto; padding: 24px; }
    .page { display: none; }
    .page.active { display: block; }
    
    /* UI Components */
    .card { background: linear-gradient(180deg, rgba(255,255,255,0.02), transparent), var(--card-bg); border-radius: 14px; padding: 20px; box-shadow: 0 10px 28px rgba(0,0,0,0.32); border: 1px solid var(--border); margin-bottom: 24px; }
    h3 { margin: 0 0 16px; font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
    .hint { color: var(--text-dim); font-size: 12px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid var(--border); padding: 12px 8px; text-align: left; }
    th { color: var(--text-dim); font-weight: 500; font-size: 12px; text-transform: uppercase; }
    input, select, textarea { width: 100%; padding: 10px; margin: 8px 0; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-family: inherit; font-size: 14px; }
    input:focus, select:focus, textarea:focus { outline: none; border-color: var(--primary); }
    .btn { padding: 8px 16px; border-radius: 6px; border: none; background: var(--primary); color: #fff; cursor: pointer; font-weight: 600; transition: 0.2s; display: inline-flex; align-items: center; justify-content: center; gap: 6px; font-size: 13px; }
    .btn:hover { background: var(--primary-hover); }
    .btn-outline { background: transparent; border: 1px solid var(--border); color: var(--text); }
    .btn-outline:hover { background: var(--border); }
    .btn-danger { background: var(--danger); }
    .btn-sm { padding: 4px 10px; font-size: 12px; }
    .btn-group { display: flex; gap: 6px; flex-wrap: wrap; }
    .status-tag { padding: 3px 8px; border-radius: 4px; font-size: 11px; font-weight: 700; display: inline-block; }
    .status-ok { background: rgba(79,212,139,.2); color: #9bf0c2; }
    .status-fail { background: rgba(255,95,109,.2); color: #ffc0c7; }
    .status-disabled { background: rgba(154,167,188,.16); color: #c6cfdb; }

    /* Dashboard */
    .stats-grid { display:grid; grid-template-columns:repeat(3, minmax(0,1fr)); gap:16px; margin-bottom:16px; }
    .stats-row-2 { display:grid; grid-template-columns:repeat(2, minmax(0,1fr)); gap:16px; }
    .stat-card { background: linear-gradient(160deg, rgba(78,161,255,.16), rgba(78,161,255,.02) 45%), var(--card-strong); border:1px solid var(--border); border-radius:12px; padding:16px; }
    .stat-title { color: var(--text-dim); font-size: 12px; letter-spacing:.2px; margin-bottom:8px; }
    .stat-value { font-size: 28px; font-weight: 700; line-height:1.1; }
    .stat-sub { margin-top:8px; color: var(--text-dim); font-size:12px; }
    
    /* Modal */
    .modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.8); display: none; align-items: center; justify-content: center; z-index: 1000; }
    .modal { background: var(--card-bg); padding: 24px; border-radius: 12px; width: 100%; max-width: 500px; border: 1px solid var(--border); position: relative; }
    .modal-close { position: absolute; top: 16px; right: 16px; cursor: pointer; font-size: 20px; color: var(--text-dim); }
    
    /* Tabs */
    .tabs { display: flex; border-bottom: 1px solid var(--border); margin-bottom: 16px; }
    .tab { padding: 10px 20px; cursor: pointer; color: var(--text-dim); border-bottom: 2px solid transparent; }
    .tab.active { color: var(--primary); border-bottom-color: var(--primary); }
    .tab-content { display: none; }
    .tab-content.active { display: block; }
    
    /* Code Block */
    pre { background: #000; padding: 16px; border-radius: 8px; overflow-x: auto; font-family: monospace; font-size: 13px; color: #a5d6a7; border: 1px solid var(--border); }
    
    /* Logs */
    .log-success { color: var(--success); }
    .log-error { color: var(--danger); }

    @media (max-width: 980px) {
      .stats-grid, .stats-row-2 { grid-template-columns: 1fr; }
      .top-bar { padding: 0 14px; }
      .content-area { padding: 14px; }
      .sidebar { width: 214px; }
    }
  </style>
</head>
<body>
  <div class="sidebar">
    <div class="sidebar-header"><span>⚡</span>LongCat Admin</div>
    <div class="nav">
      <div class="nav-item active" onclick="showPage('dashboard', event)">📊 仪表盘</div>
      <div class="nav-item" onclick="showPage('logs', event)">🧾 请求日志</div>
      <div class="nav-item" onclick="showPage('accounts', event)">👥 账户管理</div>
      <div class="nav-item" onclick="showPage('endpoints', event)">🔌 API 接口</div>
      <div class="nav-item" onclick="showPage('settings', event)">⚙️ 系统设置</div>
    </div>
    <div class="sidebar-footer">
      <a class="btn btn-outline" style="width:100%" href="/admin/logout">🚪 退出登录</a>
    </div>
  </div>

  <div class="main">
    <div class="top-bar">
      <div id="pageTitle" style="font-size:18px;font-weight:600">仪表盘</div>
      <div class="btn-group">
        <button class="btn btn-outline" onclick="refresh()">🔄 刷新数据</button>
      </div>
    </div>

    <div class="content-area">
      <!-- Dashboard Page -->
      <div id="page-dashboard" class="page active">
        <div class="card">
          <h3>📈 核心统计</h3>
          <div class="stats-grid">
            <div class="stat-card">
              <div class="stat-title">总账户数量</div>
              <div class="stat-value" id="statAccTotal">0</div>
            </div>
            <div class="stat-card">
              <div class="stat-title">正常账户数量</div>
              <div class="stat-value" id="statAccOK">0</div>
            </div>
            <div class="stat-card">
              <div class="stat-title">异常账户数量</div>
              <div class="stat-value" id="statAccFail">0</div>
            </div>
          </div>
          <div class="stats-row-2">
            <div class="stat-card">
              <div class="stat-title">请求次数（日志总数）</div>
              <div class="stat-value" id="statReqCount">0</div>
              <div class="stat-sub">按当前已记录日志统计</div>
            </div>
            <div class="stat-card">
              <div class="stat-title">Tokens 总计</div>
              <div class="stat-value" id="statTokenTotal">0</div>
              <div class="stat-sub">Prompt + Completion</div>
            </div>
          </div>
        </div>
      </div>

      <div id="page-logs" class="page">
        <div class="card">
          <h3>🧾 请求日志</h3>
          <div style="overflow-x:auto">
            <table>
              <thead>
                <tr>
                  <th>时间</th>
                  <th>模型</th>
                  <th>账号</th>
                  <th>消耗 (P/C/T)</th>
                  <th>耗时</th>
                  <th>状态</th>
                </tr>
              </thead>
              <tbody id="logTable">
                <tr><td colspan="6" style="text-align:center;padding:40px" class="hint">正在加载日志...</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Accounts Page -->
      <div id="page-accounts" class="page">
        <div class="card">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
            <h3 style="margin:0">📦 账号池管理</h3>
            <div class="btn-group">
              <button class="btn" onclick="openAddModal()">➕ 添加账号</button>
              <button class="btn btn-outline" onclick="exportConfig()">📤 导出全部</button>
            </div>
          </div>
          <div style="overflow-x:auto">
            <table id="acctTable"></table>
          </div>
        </div>
      </div>

      <!-- Endpoints Page -->
      <div id="page-endpoints" class="page">
        <div class="card">
          <h3>🔌 API 基础信息</h3>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">
            <div>
              <label class="hint">OpenAI 兼容端点</label>
              <input readonly value="{{.Scheme}}://{{.Host}}/v1/chat/completions"/>
            </div>
            <div>
              <label class="hint">Claude 兼容端点</label>
              <input readonly value="{{.Scheme}}://{{.Host}}/v1/messages"/>
            </div>
          </div>
        </div>

        <div class="card">
          <h3>🛠️ API 测试工具</h3>
          <div class="row" style="display:flex;gap:16px;margin-bottom:12px">
            <div style="flex:1">
              <label class="hint">选择模型</label>
              <select id="testModel" onchange="updateTestMessagePlaceholder()">
                <option value="LongCat-Flash">LongCat-Flash (默认)</option>
                <option value="LongCat-Thinking">LongCat-Thinking (思维链)</option>
                <option value="LongCat-Search">LongCat-Search (联网搜索)</option>
                <option value="LongCat-Search-Thinking">LongCat-Search-Thinking (全能)</option>
                <option value="LongCat-DeepResearch">LongCat-DeepResearch (深度研究)</option>
                <option value="LongCat-Image">LongCat-Image (图片生成)</option>
              </select>
            </div>
            <div style="flex:1">
              <label class="hint">API Key (可选)</label>
              <input id="testKey" type="password" placeholder="如果设置了令牌请填写"/>
            </div>
          </div>
          <label class="hint">消息内容</label>
          <textarea id="testMsg" placeholder="输入测试消息..." style="min-height:80px"></textarea>
          <button class="btn" onclick="testAPI()" id="testBtn">🚀 发送请求</button>
          <div id="testResult" style="margin-top:16px;display:none">
            <label class="hint">响应结果</label>
            <pre id="testOutput"></pre>
          </div>
        </div>

        <div class="card">
          <h3>📚 集成与测试示例</h3>
          <div class="tabs">
            <div class="tab active" onclick="showTab(this, 'ex-curl')">基础问答</div>
            <div class="tab" onclick="showTab(this, 'ex-search-think')">搜索与思考</div>
            <div class="tab" onclick="showTab(this, 'ex-image')">图片生成</div>
            <div class="tab" onclick="showTab(this, 'ex-deep-research')">深度研究</div>
          </div>
          <div id="ex-curl" class="tab-content active">
            <p style="margin-top:0; color:#888;">使用默认的 <b>LongCat-Flash</b> 模型进行极速对话：</p>
            <pre>curl {{.Scheme}}://{{.Host}}/v1/chat/completions \n  -H "Content-Type: application/json" \n  -H "Authorization: Bearer {{if .UpstreamAPIKey}}{{.UpstreamAPIKey}}{{else}}sk-your-token{{end}}" \n  -d '{
    "model": "LongCat-Flash",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'</pre>
          </div>
          <div id="ex-search-think" class="tab-content">
            <p style="margin-top:0; color:#888;">使用 <b>LongCat-Search-Thinking</b> 模型开启联网搜索与深度思考链：</p>
            <pre>curl {{.Scheme}}://{{.Host}}/v1/chat/completions \n  -H "Content-Type: application/json" \n  -H "Authorization: Bearer {{if .UpstreamAPIKey}}{{.UpstreamAPIKey}}{{else}}sk-your-token{{end}}" \n  -d '{
    "model": "LongCat-Search-Thinking",
    "messages": [{"role": "user", "content": "帮我查一下最近一周的AI新闻并分析趋势"}],
    "stream": true
  }'</pre>
          </div>
          <div id="ex-image" class="tab-content">
            <p style="margin-top:0; color:#888;">使用带有 <b>Image</b> 关键字的模型触发官方图片生成智能体 (底层对应 genImage)：</p>
            <pre>curl {{.Scheme}}://{{.Host}}/v1/chat/completions \n  -H "Content-Type: application/json" \n  -H "Authorization: Bearer {{if .UpstreamAPIKey}}{{.UpstreamAPIKey}}{{else}}sk-your-token{{end}}" \n  -d '{
    "model": "LongCat-Image",
    "messages": [{"role": "user", "content": "画一只赛博朋克风格的猫"}],
    "stream": true
  }'</pre>
          </div>
          <div id="ex-deep-research" class="tab-content">
            <p style="margin-top:0; color:#888;">使用带有 <b>DeepResearch</b> 关键字的模型触发官方深度研究智能体 (底层对应 deepResearch)：</p>
            <pre>curl {{.Scheme}}://{{.Host}}/v1/chat/completions \n  -H "Content-Type: application/json" \n  -H "Authorization: Bearer {{if .UpstreamAPIKey}}{{.UpstreamAPIKey}}{{else}}sk-your-token{{end}}" \n  -d '{
    "model": "LongCat-DeepResearch",
    "messages": [{"role": "user", "content": "吸烟到底会不会致癌？帮我做个深度研究"}],
    "stream": true
  }'</pre>
          </div>
        </div>
      </div>

      <!-- Settings Page -->
      <div id="page-settings" class="page">
        <div class="grid" style="display:grid;grid-template-columns:1fr 1fr;gap:24px">
          <div class="card">
            <h3>⚙️ 系统配置</h3>
            <label class="hint">轮询策略</label>
            <select id="strategy">
              <option value="average">Average (会话绑定) - 默认</option>
              <option value="sequential">Sequential (故障转移)</option>
            </select>
            <button class="btn" onclick="saveStrategy()" style="margin-bottom:20px">💾 保存策略</button>

            <label class="hint">接口令牌 (Upstream API Key)</label>
            <div style="display:flex;gap:8px;margin:8px 0">
              <input id="upkey" placeholder="留空则不鉴权"/>
              <button class="btn btn-outline" onclick="genKey()" style="flex:shrink-0">🎲 随机</button>
            </div>
            <button class="btn" onclick="saveKey()">💾 保存令牌</button>
          </div>

          <div class="card">
            <h3>🔐 安全设置</h3>
            <label class="hint">修改管理密码</label>
            <input id="oldpwd" type="password" placeholder="当前密码"/>
            <input id="newpwd" type="password" placeholder="新密码 (至少8位)"/>
            <button class="btn btn-danger" onclick="changePwd()">🔐 修改密码</button>
            <div id="secMsg" class="msg" style="margin-top:10px"></div>
          </div>

          <div class="card">
            <h3>💾 数据备份</h3>
            <div class="hint" style="margin-bottom:12px">导出当前所有配置，或从备份文件恢复</div>
            <div class="btn-group">
              <button class="btn btn-outline" onclick="exportConfig()">📤 导出配置</button>
              <button class="btn btn-outline" onclick="document.getElementById('importFile').click()">📥 导入配置</button>
              <input type="file" id="importFile" style="display:none" onchange="importConfig(this)"/>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Add Account Modal -->
  <div id="addModal" class="modal-overlay">
    <div class="modal">
      <span class="modal-close" onclick="closeAddModal()">&times;</span>
      <h3>➕ 添加新账号</h3>
      <div class="tabs">
        <div class="tab active" onclick="showTab(this, 'tab-cookie')">粘贴 Cookie</div>
        <div class="tab" onclick="showTab(this, 'tab-manual')">手动输入</div>
        <div class="tab" onclick="showTab(this, 'tab-import')">批量导入</div>
      </div>

      <div id="tab-cookie" class="tab-content active">
        <label class="hint">账号名称</label>
        <input id="aname" placeholder="例如: Account-1"/>
        <label class="hint">备注说明</label>
        <input id="anote" placeholder="例如: 备用账号"/>
        <label class="hint">完整 Cookie 字符串</label>
        <textarea id="acookie" placeholder="粘贴从浏览器复制的完整 Cookie..." style="height:100px"></textarea>
        <button class="btn" style="width:100%;margin-top:10px" onclick="addAccount('cookie')">确认添加</button>
      </div>

      <div id="tab-manual" class="tab-content">
        <label class="hint">账号名称</label>
        <input id="m_aname" placeholder="例如: Account-1"/>
        <label class="hint">_lxsdk_cuid</label>
        <input id="m_cuid" placeholder="必填"/>
        <label class="hint">passport_token_key</label>
        <input id="m_token" placeholder="必填"/>
        <label class="hint">_lxsdk_s</label>
        <input id="m_s" placeholder="选填"/>
        <button class="btn" style="width:100%;margin-top:10px" onclick="addAccount('manual')">确认添加</button>
      </div>

      <div id="tab-import" class="tab-content">
        <p class="hint">请使用“系统设置”中的“导入配置”功能进行批量导入。</p>
        <button class="btn btn-outline" style="width:100%" onclick="showPage('settings');closeAddModal()">前往设置</button>
      </div>
      <div id="acctMsg" class="msg" style="margin-top:10px;color:var(--danger);text-align:center"></div>
    </div>
  </div>

  <!-- Edit Account Modal -->
  <div id="editModal" class="modal-overlay">
    <div class="modal">
      <span class="modal-close" onclick="closeEditModal()">&times;</span>
      <h3>✏️ 编辑账号</h3>
      <input id="edit_id" type="hidden"/>
      <label class="hint">账号名称</label>
      <input id="edit_name"/>
      <label class="hint">备注说明</label>
      <input id="edit_note"/>
      <button class="btn" style="width:100%;margin-top:10px" onclick="saveEdit()">保存修改</button>
    </div>
  </div>

  <!-- Password Change Modal (Force) -->
  <div id="pwdModal" class="modal-overlay" {{if .MustChange}}style="display:flex"{{end}}>
    <div class="modal">
      <h3>⚠️ 安全提醒</h3>
      <p>您当前正在使用初始默认密码，为了系统安全，请立即修改密码。</p>
      <input id="force_oldpwd" type="password" placeholder="当前默认密码"/>
      <input id="force_newpwd" type="password" placeholder="新密码 (至少8位)"/>
      <button class="btn btn-danger" style="width:100%;margin-top:10px" onclick="changePwdForce()">修改密码</button>
      <div id="modalMsg" class="msg" style="color:var(--danger);margin-top:10px"></div>
    </div>
  </div>

  <script>
    function esc(s) { return (s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
    
    function showPage(id, event) {
      document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
      document.getElementById('page-' + id).classList.add('active');
      document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
      if (event && event.currentTarget) event.currentTarget.classList.add('active');
      const titles = {dashboard:'仪表盘', logs:'请求日志', accounts:'账户管理', endpoints:'API 接口', settings:'系统设置'};
      document.getElementById('pageTitle').textContent = titles[id];
      if(id === 'logs') loadLogs();
    }

    function setText(id, value) {
      const el = document.getElementById(id);
      if (el) el.textContent = value;
    }

    function formatInt(n) {
      return (Number(n) || 0).toLocaleString();
    }

    function updateDashboardStats(accounts, logs) {
      const list = Array.isArray(accounts) ? accounts : [];
      const recs = Array.isArray(logs) ? logs : [];

      const total = list.length;
      let okCount = 0;
      let failCount = 0;
      for (const a of list) {
        const enabled = !!a.enabled;
        const testedFail = a.lastTest && a.lastTest.ok === false;
        if (enabled && !testedFail) okCount += 1;
        else failCount += 1;
      }

      const reqCount = recs.length;
      const tokenTotal = recs.reduce((sum, r) => sum + (Number(r.total_tokens) || 0), 0);

      setText('statAccTotal', formatInt(total));
      setText('statAccOK', formatInt(okCount));
      setText('statAccFail', formatInt(failCount));
      setText('statReqCount', formatInt(reqCount));
      setText('statTokenTotal', formatInt(tokenTotal));
    }

    function showTab(el, id) {
      const parent = el.parentElement.parentElement;
      parent.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
      el.classList.add('active');
      parent.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
      document.getElementById(id).classList.add('active');
    }

    async function refresh() {
      const res = await fetch('/admin/api/config');
      if (!res.ok) { location.href = '/admin'; return; }
      const cfg = await res.json();
      document.getElementById('strategy').value = (cfg.strategy && cfg.strategy.type) ? cfg.strategy.type : 'average';
      document.getElementById('upkey').value = cfg.upstreamApiKey || '';
      document.getElementById('testKey').value = cfg.upstreamApiKey || '';
      renderAccounts(cfg.accounts || []);
      const logs = await loadLogs();
      updateDashboardStats(cfg.accounts || [], logs || []);
    }

    function renderAccounts(list) {
      const t = document.getElementById('acctTable');
      if (list.length === 0) {
        t.innerHTML = '<tr><td colspan="5" style="text-align:center;color:var(--text-dim);padding:30px">暂无账号，请添加</td></tr>';
        return;
      }
      let html = '<tr><th>名称/备注</th><th>状态</th><th>Cookie 摘要</th><th>健康检查</th><th>操作</th></tr>';
      for (const a of list) {
        let stClass = a.enabled ? 'status-ok' : 'status-disabled';
        let stText = a.enabled ? '启用' : '禁用';
        let testText = a.lastTest ? (a.lastTest.ok ? '正常' : '失败') : '未测试';
        let testClass = a.lastTest ? (a.lastTest.ok ? 'status-ok' : 'status-fail') : '';
        
        html += '<tr>' +
          '<td><div style="font-weight:600">' + esc(a.name) + '</div><div class="hint">' + esc(a.note || a.id) + '</div></td>' +
          '<td><span class="status-tag ' + stClass + '">' + stText + '</span></td>' +
          '<td class="hint" style="font-family:monospace;font-size:11px">' +
            'cuid: ' + esc(a.cookies._lxsdk_cuid) + '<br/>' +
            'token: ' + esc(a.cookies.passport_token_key) +
          '</td>' +
          '<td><span class="status-tag ' + testClass + '">' + testText + '</span><br/><button class="btn btn-outline btn-sm" style="margin-top:4px" onclick="testAcc(\'' + esc(a.id) + '\')">测试</button></td>' +
          '<td><div class="btn-group">' +
            '<button class="btn btn-outline btn-sm" onclick="openEditModal(\''+esc(a.id)+'\',\''+esc(a.name)+'\',\''+esc(a.note)+'\')">编辑</button>' +
            '<button class="btn btn-outline btn-sm" onclick="toggleAcc(\'' + esc(a.id) + '\',' + (!a.enabled) + ')">' + (a.enabled ? '禁用' : '启用') + '</button>' +
            '<button class="btn btn-danger btn-sm" onclick="delAcc(\'' + esc(a.id) + '\')">删除</button>' +
          '</div></td>' +
        '</tr>';
      }
      t.innerHTML = html;
    }

    async function loadLogs() {
      try {
        const res = await fetch('/admin/api/records');
        if(!res.ok) return [];
        const logs = await res.json();
        const t = document.getElementById('logTable');
        if(!logs || logs.length === 0) {
          t.innerHTML = '<tr><td colspan="6" style="text-align:center;padding:40px" class="hint">暂无调用记录</td></tr>';
          return [];
        }
        t.innerHTML = logs.map(l => 
          '<tr>' +
            '<td class="hint">' + new Date(l.timestamp).toLocaleString() + '</td>' +
            '<td>' + esc(l.model) + '</td>' +
            '<td><div style="font-weight:600">' + esc(l.account_name) + '</div><div class="hint">' + esc(l.account_id) + '</div></td>' +
            '<td class="hint">' + l.prompt_tokens + '/' + l.completion_tokens + '/' + l.total_tokens + '</td>' +
            '<td class="hint">' + l.latency.toFixed(2) + 's</td>' +
            '<td><span class="status-tag ' + (l.status === 'success' ? 'status-ok' : 'status-fail') + '">' + l.status.toUpperCase() + '</span></td>' +
          '</tr>'
        ).join('');
        return logs;
      } catch (e) { }
      return [];
    }

    function openAddModal() { document.getElementById('addModal').style.display = 'flex'; }
    function closeAddModal() { document.getElementById('addModal').style.display = 'none'; }

    async function addAccount(type) {
      let data = {};
      if(type === 'cookie') {
        data = {
          name: document.getElementById('aname').value,
          note: document.getElementById('anote').value,
          cookies: document.getElementById('acookie').value
        };
      } else {
        data = {
          name: document.getElementById('m_aname').value,
          cookies: {
            _lxsdk_cuid: document.getElementById('m_cuid').value,
            passport_token_key: document.getElementById('m_token').value,
            _lxsdk_s: document.getElementById('m_s').value
          }
        };
      }
      const res = await fetch('/admin/api/account', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(data)
      });
      if(res.ok) {
        document.getElementById('aname').value = '';
        document.getElementById('anote').value = '';
        document.getElementById('acookie').value = '';
        document.getElementById('m_aname').value = '';
        document.getElementById('m_cuid').value = '';
        document.getElementById('m_token').value = '';
        document.getElementById('m_s').value = '';
        document.getElementById('acctMsg').textContent = '';
        closeAddModal();
        refresh();
      } else {
        const j = await res.json().catch(() => ({error: '添加失败'}));
        document.getElementById('acctMsg').textContent = j.error || '添加失败';
      }
    }

    function openEditModal(id, name, note) {
      document.getElementById('edit_id').value = id;
      document.getElementById('edit_name').value = name;
      document.getElementById('edit_note').value = note;
      document.getElementById('editModal').style.display = 'flex';
    }
    function closeEditModal() { document.getElementById('editModal').style.display = 'none'; }

    async function saveEdit() {
      const id = document.getElementById('edit_id').value;
      const res = await fetch('/admin/api/account/' + id, {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          name: document.getElementById('edit_name').value,
          note: document.getElementById('edit_note').value
        })
      });
      if(res.ok) { closeEditModal(); refresh(); }
      else alert('保存失败');
    }

    async function toggleAcc(id, enabled) {
      await fetch('/admin/api/account/' + id, {
        method: 'PUT',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({enabled: enabled})
      });
      refresh();
    }

    async function delAcc(id) {
      if (!confirm('确认删除该账号？')) return;
      await fetch('/admin/api/account/' + id, {method: 'DELETE'});
      refresh();
    }

    async function testAcc(id) {
      await fetch('/admin/api/account/' + id + '/test', {method: 'POST'});
      refresh();
    }

    function updateTestMessagePlaceholder() {
      const model = document.getElementById('testModel').value;
      const msg = document.getElementById('testMsg');
      if (model.includes('Image')) {
        msg.placeholder = "输入提示词，例如：画一只在赛博朋克城市里的猫";
        msg.value = "画一只在赛博朋克城市里的猫";
      } else if (model.includes('DeepResearch')) {
        msg.placeholder = "输入研究主题，例如：吸烟到底会不会致癌";
        msg.value = "吸烟到底会不会致癌？帮我做个深度研究";
      } else {
        msg.placeholder = "输入测试消息...";
        msg.value = "你好";
      }
    }

    async function testAPI() {
      const btn = document.getElementById('testBtn');
      const out = document.getElementById('testOutput');
      const resDiv = document.getElementById('testResult');
      btn.disabled = true;
      btn.textContent = '⏳ 请求中...';
      resDiv.style.display = 'block';
      out.textContent = '正在发送请求...';

      try {
        const res = await fetch('/v1/chat/completions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + document.getElementById('testKey').value
          },
          body: JSON.stringify({
            model: document.getElementById('testModel').value,
            messages: [{role: 'user', content: document.getElementById('testMsg').value}],
            stream: false
          })
        });
        if (!res.ok) {
          const text = await res.text();
          out.textContent = '请求失败 (' + res.status + '):\n' + text;
        } else {
          const j = await res.json();
          out.textContent = JSON.stringify(j, null, 2);
        }
      } catch (e) {
        out.textContent = '请求失败: ' + (e && e.message ? e.message : String(e));
      } finally {
        btn.disabled = false;
        btn.textContent = '🚀 发送请求';
        refresh(); // Refresh to see logs
      }
    }

    async function saveStrategy() {
      const type = document.getElementById('strategy').value;
      await fetch('/admin/api/strategy', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({type: type})
      });
      alert('策略已保存');
    }

    async function saveKey() {
      const key = document.getElementById('upkey').value;
      await fetch('/admin/api/upstream-key', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({key: key})
      });
      alert('令牌已保存');
    }

    function genKey() {
      const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
      let key = 'sk-';
      for (let i = 0; i < 32; i++) key += chars.charAt(Math.floor(Math.random() * chars.length));
      document.getElementById('upkey').value = key;
    }

    async function changePwd() {
      const oldPwd = document.getElementById('oldpwd').value;
      const newPwd = document.getElementById('newpwd').value;
      const res = await fetch('/admin/api/password', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({old: oldPwd, new: newPwd})
      });
      if (res.ok) { alert('密码已修改，请重新登录'); location.href = '/admin'; }
      else { const j = await res.json(); alert(j.error || '修改失败'); }
    }

    async function changePwdForce() {
      const oldPwd = document.getElementById('force_oldpwd').value;
      const newPwd = document.getElementById('force_newpwd').value;
      const msg = document.getElementById('modalMsg');
      const res = await fetch('/admin/api/password', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({old: oldPwd, new: newPwd})
      });
      if (res.ok) { alert('密码已修改，请重新登录'); location.href = '/admin'; }
      else { const j = await res.json(); msg.textContent = j.error || '修改失败'; }
    }

    function exportConfig() { window.open('/admin/api/export'); }
    async function importConfig(input) {
      if (!input.files || !input.files[0]) return;
      const formData = new FormData();
      formData.append('file', input.files[0]);
      const res = await fetch('/admin/api/import', {method: 'POST', body: formData});
      if (res.ok) { alert('导入成功'); location.reload(); }
      else alert('导入失败');
    }

    refresh();
    setInterval(refresh, 10000); // Auto refresh dashboard, logs and accounts every 10s
  </script>
</body>
</html>`
