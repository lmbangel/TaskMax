<script>
  import { onMount, onDestroy } from 'svelte'
  import { tasks } from './stores/tasks.js'
  import { timer } from './stores/timer.js'
  import TaskList from './lib/TaskList.svelte'
  import TaskForm from './lib/TaskForm.svelte'
  import PomodoroTimer from './lib/PomodoroTimer.svelte'
  import Settings from './lib/Settings.svelte'
  import StatsPanel from './lib/StatsPanel.svelte'
  import {
    GetConfig,
    SaveConfig,
    GetTodayStats,
    GetSessionsForTask
  } from '../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

  const FILTERS = [
    { key: 'all', label: 'All' },
    { key: 'todo', label: 'Todo' },
    { key: 'in_progress', label: 'In Progress' },
    { key: 'done', label: 'Done' }
  ]

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
    filter === 'all' ? $tasks : $tasks.filter((t) => t.Status === filter)

  // Keep selection valid as the list changes.
  $: if (selectedId && !$tasks.some((t) => t.ID === selectedId)) {
    selectedId = null
  }

  async function applyTheme(theme) {
    document.documentElement.dataset.theme = theme || 'cosy'
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
    selectedId = e.detail.ID
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
    applyTheme(config.app.theme)
    settingsOpen = false
    loadStats()
  }

  let statsInterval = null

  onMount(async () => {
    try {
      config = await GetConfig()
      applyTheme(config.app.theme)
    } catch (e) {
      applyTheme('cosy')
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
  })

  onDestroy(() => {
    EventsOff('pomodoro:complete')
    if (statsInterval) clearInterval(statsInterval)
    timer.stop()
  })
</script>

<div class="app">
  <!-- Left: task list -->
  <aside class="sidebar">
    <div class="brand">
      <span class="logo">🍅</span>
      <span class="name">TaskMax</span>
      <button class="btn btn-ghost settings-btn" title="Settings" on:click={() => (settingsOpen = true)}>
        ⚙️
      </button>
    </div>

    <button class="btn btn-accent new-btn" on:click={newTask}>+ New task</button>

    <div class="filters">
      {#each FILTERS as f}
        <button class="filter" class:active={filter === f.key} on:click={() => (filter = f.key)}>
          {f.label}
        </button>
      {/each}
    </div>

    <TaskList
      tasks={filtered}
      {selectedId}
      on:select={selectTask}
      on:statusChange={changeStatus}
      on:reorder={reorderTasks}
    />
  </aside>

  <!-- Centre: detail + timer -->
  <main class="centre">
    {#if selectedTask}
      <div class="detail card">
        <div class="detail-head">
          <div>
            <h1>{selectedTask.Title}</h1>
            <div class="detail-meta">
              <span class="badge {selectedTask.Priority}">{selectedTask.Priority}</span>
              <span class="status-pill">{selectedTask.Status.replace('_', ' ')}</span>
              {#if selectedTask.DueDate}
                <span class="due">📅 {String(selectedTask.DueDate).slice(0, 10)}</span>
              {/if}
            </div>
          </div>
          <div class="detail-actions">
            <button class="btn btn-ghost" on:click={editTask}>✎ Edit</button>
            <button class="btn btn-ghost danger" on:click={deleteSelected}>🗑</button>
          </div>
        </div>
        {#if selectedTask.Description}
          <p class="desc">{selectedTask.Description}</p>
        {/if}
      </div>
    {:else}
      <div class="detail card placeholder">
        <span class="emoji">🍅</span>
        <h1>Focus time</h1>
        <p>Select a task on the left, or just start the timer to begin a session.</p>
      </div>
    {/if}

    <PomodoroTimer task={selectedTask} {config} />
  </main>

  <!-- Right: stats -->
  <aside class="rightbar">
    <StatsPanel {stats} {sessions} task={selectedTask} />
  </aside>
</div>

<TaskForm open={formOpen} task={formTask} on:save={saveTask} on:close={() => (formOpen = false)} />
<Settings open={settingsOpen} {config} on:save={saveSettings} on:close={() => (settingsOpen = false)} />

<style>
  .app {
    display: grid;
    grid-template-columns: 320px 1fr 300px;
    height: 100vh;
    gap: 1rem;
    padding: 1rem;
  }

  .sidebar,
  .rightbar {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    min-height: 0;
  }
  .sidebar {
    background: var(--surface);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow);
    padding: 1rem;
  }
  .rightbar {
    padding: 0.25rem;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.55rem;
  }
  .brand .logo {
    font-size: 1.3rem;
  }
  .brand .name {
    font-weight: 800;
    font-size: 1.15rem;
    letter-spacing: 0.01em;
  }
  .settings-btn {
    margin-left: auto;
    padding: 0.35rem 0.5rem;
    font-size: 1rem;
  }
  .new-btn {
    width: 100%;
    padding: 0.7rem;
  }

  .filters {
    display: flex;
    gap: 0.3rem;
    flex-wrap: wrap;
  }
  .filter {
    flex: 1 1 auto;
    padding: 0.4rem 0.5rem;
    border-radius: var(--radius-input);
    background: var(--surface-2);
    color: var(--text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    white-space: nowrap;
  }
  .filter.active {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .centre {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-height: 0;
    overflow-y: auto;
  }
  .detail {
    padding: 1.4rem 1.6rem;
  }
  .detail.placeholder {
    text-align: center;
    color: var(--text-muted);
    padding: 2.4rem 1.6rem;
  }
  .detail.placeholder .emoji {
    font-size: 2.2rem;
  }
  .detail.placeholder h1 {
    margin: 0.5rem 0 0.3rem;
  }
  .detail-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }
  .detail-head h1 {
    margin: 0 0 0.6rem;
    font-size: 1.4rem;
  }
  .detail-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .status-pill {
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--text-muted);
    background: var(--surface-2);
    padding: 0.2rem 0.55rem;
    border-radius: 999px;
    text-transform: capitalize;
  }
  .due {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .detail-actions {
    display: flex;
    gap: 0.4rem;
    flex: 0 0 auto;
  }
  .detail-actions .danger:hover {
    color: var(--danger);
  }
  .desc {
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.55;
    white-space: pre-wrap;
  }

  @media (max-width: 1040px) {
    .app {
      grid-template-columns: 280px 1fr;
    }
    .rightbar {
      display: none;
    }
  }
</style>
