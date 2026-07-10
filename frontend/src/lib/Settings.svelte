<script>
  import { createEventDispatcher } from 'svelte'
  import {
    TestConnection,
    GetAppVersion,
    GetLaunchOnStartup,
    SetLaunchOnStartup,
    ExportData,
    ImportData
  } from '../../wailsjs/go/main/App'
  import { ACCENTS, DEFAULT_ACCENT } from './accents.js'

  let version = ''
  $: if (open && !version) {
    GetAppVersion()
      .then((v) => (version = v))
      .catch(() => {})
  }

  // Launch-on-startup lives in the OS (registry), not config.yaml.
  let launchOnStartup = false
  let launchLoaded = false
  $: if (open && !launchLoaded) {
    launchLoaded = true
    GetLaunchOnStartup()
      .then((v) => (launchOnStartup = v))
      .catch(() => {})
  }

  export let open = false
  export let config = null

  const dispatch = createEventDispatcher()

  let draft = null
  let testState = 'idle' // idle | testing | ok | fail
  let testMsg = ''

  // Deep-ish clone whenever the panel opens so edits are cancellable.
  $: if (open && config && !draft) {
    draft = {
      database: { ...config.database },
      pomodoro: { ...config.pomodoro },
      app: { accent: DEFAULT_ACCENT, ...config.app },
      mcp: { enabled: true, port: 7823, ...config.mcp }
    }
    testState = 'idle'
    testMsg = ''
  }

  // Saving updates `config` while the panel is still open, which reseeds the
  // draft above; if that draft survived the close, the next open would show
  // (and re-save) stale values. Always drop it once the panel is closed.
  $: if (!open && draft) {
    draft = null
  }

  function close() {
    draft = null
    launchLoaded = false
    dispatch('close')
  }

  function save() {
    SetLaunchOnStartup(launchOnStartup).catch(() => {})
    launchLoaded = false
    dispatch('save', draft)
    draft = null
  }

  // ----- Data export / import -----
  let dataState = 'idle' // idle | busy | ok | fail
  let dataMsg = ''

  async function doExport() {
    dataState = 'busy'
    dataMsg = ''
    try {
      const path = await ExportData()
      if (path) {
        dataState = 'ok'
        dataMsg = `Exported to ${path}`
      } else {
        dataState = 'idle' // dialog cancelled
      }
    } catch (e) {
      dataState = 'fail'
      dataMsg = (e && e.message) || String(e)
    }
  }

  async function doImport(mode) {
    dataState = 'busy'
    dataMsg = ''
    try {
      const res = await ImportData(mode)
      if (res.canceled) {
        dataState = 'idle'
      } else {
        dataState = 'ok'
        dataMsg = `Imported ${res.tasks_imported} tasks and ${res.sessions_imported} sessions`
      }
    } catch (e) {
      dataState = 'fail'
      dataMsg = (e && e.message) || String(e)
    }
  }

  async function testConnection() {
    testState = 'testing'
    testMsg = ''
    try {
      await TestConnection(draft.database.type, draft.database.dsn)
      testState = 'ok'
      testMsg = 'Connection successful'
    } catch (e) {
      testState = 'fail'
      testMsg = (e && e.message) || String(e)
    }
  }
</script>

{#if open && draft}
  <div class="scrim" on:click={close} role="presentation"></div>
  <aside class="panel">
    <header>
      <h2>⚙️ Settings</h2>
      <button class="btn btn-ghost close" on:click={close}>✕</button>
    </header>

    <div class="fields">
      <!-- Database -->
      <section>
        <h3>Database</h3>
        <div class="row">
          <div>
            <label for="db-type">Type</label>
            <select id="db-type" bind:value={draft.database.type} on:change={() => (testState = 'idle')}>
              <option value="sqlite">SQLite (local)</option>
              <option value="postgres">PostgreSQL</option>
              <option value="mysql">MySQL</option>
            </select>
          </div>
          <div>
            <label for="db-dsn">
              {draft.database.type === 'sqlite' ? 'File path' : 'Connection string (DSN)'}
            </label>
            <input
              id="db-dsn"
              bind:value={draft.database.dsn}
              on:input={() => (testState = 'idle')}
              placeholder={draft.database.type === 'sqlite'
                ? 'tasks.db'
                : 'host=… user=… dbname=…'}
            />
          </div>
        </div>
        <div class="test-row">
          <button class="btn" on:click={testConnection} disabled={testState === 'testing'}>
            {testState === 'testing' ? 'Testing…' : 'Test connection'}
          </button>
          {#if testState === 'ok'}
            <span class="test ok">✓ {testMsg}</span>
          {:else if testState === 'fail'}
            <span class="test fail">✕ {testMsg}</span>
          {/if}
        </div>
        <p class="note">Changing the database driver takes effect after an app restart.</p>
      </section>

      <!-- Pomodoro -->
      <section>
        <h3>Pomodoro</h3>
        <label class="slider">
          <span class="slabel">Work duration <b>{draft.pomodoro.work_duration} min</b></span>
          <input type="range" min="5" max="60" step="5" bind:value={draft.pomodoro.work_duration} />
        </label>
        <label class="slider">
          <span class="slabel">Short break <b>{draft.pomodoro.short_break} min</b></span>
          <input type="range" min="1" max="30" step="1" bind:value={draft.pomodoro.short_break} />
        </label>
        <label class="slider">
          <span class="slabel">Long break <b>{draft.pomodoro.long_break} min</b></span>
          <input type="range" min="5" max="45" step="5" bind:value={draft.pomodoro.long_break} />
        </label>
        <label class="slider">
          <span class="slabel">Sessions before long break <b>{draft.pomodoro.sessions_before_long}</b></span>
          <input type="range" min="2" max="8" step="1" bind:value={draft.pomodoro.sessions_before_long} />
        </label>
        <label class="slider">
          <span class="slabel">Daily goal <b>{draft.pomodoro.daily_goal} sessions</b></span>
          <input type="range" min="1" max="16" step="1" bind:value={draft.pomodoro.daily_goal} />
        </label>
        <label class="switch">
          <input type="checkbox" bind:checked={draft.pomodoro.sound} />
          <span>Chime when a session ends</span>
        </label>
      </section>

      <!-- Appearance -->
      <section>
        <h3>Appearance</h3>
        <span class="fieldlabel">Mode</span>
        <div class="themes">
          {#each ['cosy', 'dark', 'light'] as t}
            <button
              class="theme-btn"
              class:active={draft.app.theme === t}
              on:click={() => (draft.app.theme = t)}
            >
              {t}
            </button>
          {/each}
        </div>

        <span class="fieldlabel">Accent</span>
        <div class="themes">
          {#each Object.entries(ACCENTS) as [key, a]}
            <button
              class="theme-btn"
              class:active={draft.app.accent === key}
              on:click={() => (draft.app.accent = key)}
            >
              {a.emoji} {a.label}
            </button>
          {/each}
        </div>

        <label class="switch">
          <input type="checkbox" bind:checked={draft.app.minimize_to_tray} />
          <span>Hide window on close (minimize to tray)</span>
        </label>
        <label class="switch">
          <input type="checkbox" bind:checked={launchOnStartup} />
          <span>Launch TaskMax when Windows starts</span>
        </label>
      </section>

      <!-- Data -->
      <section>
        <h3>Data</h3>
        <div class="data-row">
          <button class="btn" on:click={doExport} disabled={dataState === 'busy'}>
            ⬇ Export…
          </button>
          <button class="btn" on:click={() => doImport('merge')} disabled={dataState === 'busy'}>
            ⬆ Import…
          </button>
          <button class="btn" on:click={() => doImport('replace')} disabled={dataState === 'busy'}>
            ♻ Restore…
          </button>
        </div>
        {#if dataState === 'ok'}
          <p class="test ok data-msg">✓ {dataMsg}</p>
        {:else if dataState === 'fail'}
          <p class="test fail data-msg">✕ {dataMsg}</p>
        {/if}
        <p class="note">
          Export saves all tasks and session history to a JSON file.
          Import adds a backup's tasks to the board; Restore replaces everything with the backup.
        </p>
      </section>

      <!-- Agents -->
      <section>
        <h3>Agents (MCP)</h3>
        <label class="switch">
          <input type="checkbox" bind:checked={draft.mcp.enabled} />
          <span>Let coding agents manage tasks (MCP server)</span>
        </label>
        <label class="switch">
          <input type="checkbox" bind:checked={draft.app.agent_notifications} />
          <span>Notify when agents create or complete tasks</span>
        </label>
        <p class="note">
          Serves MCP on http://localhost:{draft.mcp.port}/mcp — local connections only.
          Connect Claude Code with:<br />
          <code>claude mcp add --transport http taskmax http://localhost:{draft.mcp.port}/mcp</code><br />
          Changing this takes effect after an app restart.
        </p>
      </section>
    </div>

    <footer>
      {#if version}
        <span class="app-version">TaskMax {version}</span>
      {/if}
      <button class="btn btn-ghost" on:click={close}>Cancel</button>
      <button class="btn btn-accent" on:click={save}>Save settings</button>
    </footer>
  </aside>
{/if}

<style>
  .scrim {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: 40;
    animation: fade 0.15s ease;
  }
  .panel {
    position: fixed;
    top: 0;
    right: 0;
    height: 100vh;
    width: min(460px, 94vw);
    background: var(--surface);
    box-shadow: var(--shadow-lg);
    z-index: 41;
    display: flex;
    flex-direction: column;
    animation: slide 0.22s cubic-bezier(0.22, 1, 0.36, 1);
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 1.5rem 0.75rem;
  }
  header h2 {
    margin: 0;
    font-size: 1.15rem;
  }
  .close {
    padding: 0.35rem 0.6rem;
  }
  .fields {
    flex: 1 1 auto;
    overflow-y: auto;
    padding: 0.5rem 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  section h3 {
    margin: 0 0 0.75rem;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
  }
  .row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.9rem;
  }
  .test-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.75rem;
  }
  .test {
    font-size: 0.78rem;
    font-weight: 600;
  }
  .test.ok {
    color: var(--success);
  }
  .test.fail {
    color: var(--danger);
  }
  .data-row {
    display: flex;
    gap: 0.5rem;
  }
  .data-row .btn {
    flex: 1;
    padding: 0.5rem 0.4rem;
    font-size: 0.78rem;
    white-space: nowrap;
  }
  .data-msg {
    margin: 0.5rem 0 0;
    word-break: break-all;
  }
  .note {
    font-size: 0.72rem;
    color: var(--text-faint);
    margin: 0.6rem 0 0;
  }
  .slider {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 0.9rem;
  }
  .slabel {
    display: flex;
    justify-content: space-between;
    font-size: 0.78rem;
    color: var(--text-muted);
  }
  .slabel b {
    color: var(--accent-ink);
  }
  .fieldlabel {
    display: block;
    font-size: 0.78rem;
    color: var(--text-muted);
    margin-bottom: 0.35rem;
  }
  input[type='range'] {
    width: 100%;
    accent-color: var(--accent);
    padding: 0;
    background: transparent;
  }
  .themes {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }
  .theme-btn {
    flex: 1;
    padding: 0.55rem;
    border-radius: var(--radius-input);
    background: var(--surface-2);
    color: var(--text-muted);
    text-transform: capitalize;
    font-weight: 600;
    font-size: 0.85rem;
  }
  .theme-btn.active {
    background: var(--accent);
    color: var(--on-accent);
  }
  .switch {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    text-transform: none;
    color: var(--text);
    font-size: 0.85rem;
    cursor: pointer;
    margin-bottom: 0.6rem;
  }
  .switch input {
    width: auto;
    accent-color: var(--accent);
  }
  footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.6rem;
    padding: 1rem 1.5rem 1.25rem;
    border-top: 1px solid var(--surface-2);
  }
  .app-version {
    margin-right: auto;
    font-size: 0.7rem;
    color: var(--text-faint);
  }
  @keyframes slide {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
