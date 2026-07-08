<script>
  import { onMount, onDestroy } from 'svelte'
  import { tasks } from './stores/tasks.js'
  import { timer } from './stores/timer.js'
  import { accent, mascot } from './stores/appearance.js'
  import { DEFAULT_ACCENT } from './lib/accents.js'
  import TaskList from './lib/TaskList.svelte'
  import TaskForm from './lib/TaskForm.svelte'
  import PomodoroTimer from './lib/PomodoroTimer.svelte'
  import Settings from './lib/Settings.svelte'
  import StatsPanel from './lib/StatsPanel.svelte'
  import {
    GetConfig,
    SaveConfig,
    GetTodayStats,
    GetSessionsForTask,
    CheckForUpdate
  } from '../wailsjs/go/main/App'
  import {
    EventsOn,
    EventsOff,
    WindowMinimise,
    WindowHide,
    BrowserOpenURL,
    Quit
  } from '../wailsjs/runtime/runtime'

  const TABS = [
    { key: 'focus', label: 'Focus' },
    { key: 'tasks', label: 'Tasks' },
    { key: 'stats', label: 'Stats' }
  ]

  const FILTERS = [
    { key: 'all', label: 'All' },
    { key: 'todo', label: 'Todo' },
    { key: 'in_progress', label: 'Doing' },
    { key: 'done', label: 'Done' }
  ]

  let tab = 'focus'
  let filter = 'all'
  let selectedId = null
  let config = null

  let formOpen = false
  let formTask = null
  let settingsOpen = false

  let stats = { sessions_completed: 0, work_sessions: 0, total_focus_minutes: 0 }
  let sessions = []

  $: selectedTask = $tasks.find((t) => t.ID === selectedId) || null
  $: filtered =
    filter === 'all' ? $tasks : $tasks.filter((t) => t.status === filter)

  // Keep selection valid as the list changes.
  $: if (selectedId && !$tasks.some((t) => t.ID === selectedId)) {
    selectedId = null
  }

  function applyAppearance(theme, accentKey) {
    document.documentElement.dataset.theme = theme || 'cosy'
    document.documentElement.dataset.accent = accentKey || DEFAULT_ACCENT
    accent.set(accentKey || DEFAULT_ACCENT)
  }

  async function loadStats() {
    try {
      stats = await GetTodayStats()
    } catch (e) {
      /* runtime not ready */
    }
  }

  async function loadSessions() {
    if (!selectedTask) {
      sessions = []
      return
    }
    try {
      sessions = (await GetSessionsForTask(selectedTask.ID)) || []
    } catch (e) {
      sessions = []
    }
  }

  function selectTask(e) {
    selectedId = selectedId === e.detail.ID ? null : e.detail.ID
    loadSessions()
  }

  function newTask() {
    formTask = null
    formOpen = true
  }

  function editTask() {
    if (!selectedTask) return
    formTask = selectedTask
    formOpen = true
  }

  async function saveTask(e) {
    const payload = e.detail
    if (payload.ID) {
      await tasks.update(payload)
      selectedId = payload.ID
    } else {
      const created = await tasks.create(payload)
      if (created && created.ID) selectedId = created.ID
    }
    loadSessions()
  }

  async function changeStatus(e) {
    await tasks.update(e.detail)
  }

  async function reorderTasks(e) {
    await tasks.reorder(e.detail)
  }

  async function deleteSelected() {
    if (!selectedTask) return
    await tasks.remove(selectedTask.ID)
    selectedId = null
    sessions = []
  }

  async function saveSettings(e) {
    config = e.detail
    await SaveConfig(config)
    applyAppearance(config.app.theme, config.app.accent)
    settingsOpen = false
    loadStats()
  }

  // With the tray icon in place, close can safely hide to the tray when the
  // user has minimize_to_tray enabled; the tray restores or quits the app.
  function closeApp() {
    if (config?.app?.minimize_to_tray) WindowHide()
    else Quit()
  }

  // ----- Update check -----
  let update = null // UpdateInfo from Go when a newer release exists

  async function checkForUpdate() {
    try {
      const u = await CheckForUpdate()
      // Stay quiet about versions the user already dismissed.
      if (u?.available && localStorage.getItem('update-dismissed') !== u.latest_version) {
        update = u
      }
    } catch (e) {
      /* offline or rate-limited — try again next interval */
    }
  }

  function dismissUpdate() {
    if (update) localStorage.setItem('update-dismissed', update.latest_version)
    update = null
  }

  let statsInterval = null
  let updateInterval = null

  onMount(async () => {
    try {
      config = await GetConfig()
      applyAppearance(config.app.theme, config.app.accent)
    } catch (e) {
      applyAppearance('cosy', DEFAULT_ACCENT)
    }
    await tasks.refresh()
    await loadStats()

    // A completed session updates task pomodoro counts and today's stats.
    EventsOn('pomodoro:complete', () => {
      tasks.refresh()
      loadStats()
      loadSessions()
    })

    statsInterval = setInterval(loadStats, 30000)

    checkForUpdate()
    updateInterval = setInterval(checkForUpdate, 24 * 60 * 60 * 1000)
  })

  onDestroy(() => {
    EventsOff('pomodoro:complete')
    if (statsInterval) clearInterval(statsInterval)
    if (updateInterval) clearInterval(updateInterval)
    timer.stop()
  })
</script>

<div class="widget">
  <!-- Custom titlebar: the window is frameless, so this is the drag handle. -->
  <header class="titlebar" style="--wails-draggable: drag">
    <span class="logo">{$mascot}</span>
    <span class="name">TaskMax</span>
    <div class="win-controls" style="--wails-draggable: no-drag">
      <button class="win-btn" title="Settings" on:click={() => (settingsOpen = true)}>⚙</button>
      <button class="win-btn" title="Minimize" on:click={WindowMinimise}>–</button>
      <button class="win-btn close" title="Close" on:click={closeApp}>✕</button>
    </div>
  </header>

  {#if update}
    <div class="update-banner">
      <span class="ub-text">{update.latest_version} is available</span>
      <button class="ub-get" on:click={() => BrowserOpenURL(update.url)}>Get update</button>
      <button class="ub-close" title="Dismiss" on:click={dismissUpdate}>✕</button>
    </div>
  {/if}

  <nav class="tabs">
    {#each TABS as t}
      <button class="tab" class:active={tab === t.key} on:click={() => (tab = t.key)}>
        {t.label}
      </button>
    {/each}
  </nav>

  <main class="content">
    {#if tab === 'focus'}
      {#if selectedTask}
        <div class="focus-task card">
          <span class="ft-label">Focusing on</span>
          <span class="ft-title">{selectedTask.title}</span>
          <button class="btn btn-ghost ft-clear" title="Clear task" on:click={() => (selectedId = null)}>✕</button>
        </div>
      {/if}
      <PomodoroTimer task={selectedTask} {config} />
    {:else if tab === 'tasks'}
      <button class="btn btn-accent new-btn" on:click={newTask}>+ New task</button>

      <div class="filters">
        {#each FILTERS as f}
          <button class="filter" class:active={filter === f.key} on:click={() => (filter = f.key)}>
            {f.label}
          </button>
        {/each}
      </div>

      {#if selectedTask}
        <div class="detail card">
          <div class="detail-top">
            <span class="detail-title">{selectedTask.title}</span>
            <span class="badge {selectedTask.priority}">{selectedTask.priority}</span>
          </div>
          {#if selectedTask.description}
            <p class="desc">{selectedTask.description}</p>
          {/if}
          <div class="detail-meta">
            <span class="status-pill">{selectedTask.status.replace('_', ' ')}</span>
            {#if selectedTask.due_date}
              <span class="due">📅 {String(selectedTask.due_date).slice(0, 10)}</span>
            {/if}
          </div>
          <div class="detail-actions">
            <button class="btn btn-accent" on:click={() => (tab = 'focus')}>{$mascot} Focus</button>
            <button class="btn btn-ghost" on:click={editTask}>✎ Edit</button>
            <button class="btn btn-ghost danger" on:click={deleteSelected}>🗑</button>
          </div>
        </div>
      {/if}

      <TaskList
        tasks={filtered}
        {selectedId}
        on:select={selectTask}
        on:statusChange={changeStatus}
        on:reorder={reorderTasks}
      />
    {:else}
      <StatsPanel {stats} {sessions} task={selectedTask} />
    {/if}
  </main>
</div>

<TaskForm open={formOpen} task={formTask} on:save={saveTask} on:close={() => (formOpen = false)} />
<Settings open={settingsOpen} {config} on:save={saveSettings} on:close={() => (settingsOpen = false)} />

<style>
  .widget {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }

  /* ----- Titlebar ----- */
  .titlebar {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.55rem 0.6rem 0.55rem 0.85rem;
    flex: 0 0 auto;
    user-select: none;
    cursor: grab;
  }
  .titlebar .logo {
    font-size: 1.05rem;
  }
  .titlebar .name {
    font-weight: 800;
    font-size: 0.95rem;
    letter-spacing: 0.01em;
  }
  .win-controls {
    margin-left: auto;
    display: flex;
    gap: 0.15rem;
  }
  .win-btn {
    width: 28px;
    height: 26px;
    border-radius: 7px;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.85rem;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .win-btn:hover {
    background: var(--surface-2);
    color: var(--text);
  }
  .win-btn.close:hover {
    background: var(--danger);
    color: #fff;
  }

  /* ----- Update banner ----- */
  .update-banner {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0.75rem 0.6rem;
    padding: 0.45rem 0.4rem 0.45rem 0.75rem;
    border-radius: var(--radius-input);
    background: var(--accent-soft);
    flex: 0 0 auto;
  }
  .ub-text {
    flex: 1 1 auto;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--accent-ink);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ub-get {
    flex: 0 0 auto;
    padding: 0.3rem 0.6rem;
    border-radius: 6px;
    background: var(--accent);
    color: var(--on-accent);
    font-size: 0.72rem;
    font-weight: 700;
  }
  .ub-close {
    flex: 0 0 auto;
    width: 22px;
    height: 22px;
    border-radius: 6px;
    background: transparent;
    color: var(--accent-ink);
    font-size: 0.7rem;
    line-height: 1;
  }
  .ub-close:hover {
    background: var(--surface-2);
  }

  /* ----- Tabs ----- */
  .tabs {
    display: flex;
    gap: 0.3rem;
    padding: 0 0.75rem 0.6rem;
    flex: 0 0 auto;
  }
  .tab {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    padding: 0.5rem 0.4rem;
    border-radius: var(--radius-input);
    background: var(--surface);
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 700;
    box-shadow: var(--shadow);
  }
  .tab.active {
    background: var(--accent-soft);
    color: var(--accent-ink);
  }

  /* ----- Content ----- */
  .content {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
    padding: 0.1rem 0.75rem 0.75rem;
  }

  .focus-task {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.8rem;
  }
  .ft-label {
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-faint);
    flex: 0 0 auto;
  }
  .ft-title {
    font-weight: 600;
    font-size: 0.85rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1 1 auto;
  }
  .ft-clear {
    padding: 0.2rem 0.45rem;
    font-size: 0.75rem;
    flex: 0 0 auto;
  }

  .new-btn {
    width: 100%;
    padding: 0.65rem;
    flex: 0 0 auto;
  }

  .filters {
    display: flex;
    gap: 0.3rem;
    flex: 0 0 auto;
  }
  .filter {
    flex: 1 1 auto;
    padding: 0.4rem 0.3rem;
    border-radius: var(--radius-input);
    background: var(--surface-2);
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 600;
    white-space: nowrap;
  }
  .filter.active {
    background: var(--accent-soft);
    color: var(--accent-ink);
  }

  /* ----- Selected task detail (Tasks tab) ----- */
  .detail {
    padding: 0.8rem 0.9rem;
    flex: 0 0 auto;
  }
  .detail-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .detail-title {
    font-weight: 700;
    font-size: 0.95rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .desc {
    margin: 0.5rem 0 0;
    color: var(--text-muted);
    font-size: 0.8rem;
    line-height: 1.5;
    white-space: pre-wrap;
  }
  .detail-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    margin-top: 0.55rem;
  }
  .status-pill {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--text-muted);
    background: var(--surface-2);
    padding: 0.18rem 0.5rem;
    border-radius: 999px;
    text-transform: capitalize;
  }
  .due {
    font-size: 0.72rem;
    color: var(--text-muted);
  }
  .detail-actions {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.7rem;
  }
  .detail-actions .btn {
    padding: 0.45rem 0.7rem;
    font-size: 0.78rem;
  }
  .detail-actions .danger:hover {
    color: var(--danger);
  }
</style>
