<script>
  import { mascot } from '../stores/appearance.js'

  export let stats = { sessions_completed: 0, work_sessions: 0, total_focus_minutes: 0 }
  export let sessions = [] // sessions for the selected task
  export let task = null

  // Work sessions use the accent mascot as their icon.
  const TYPE_LABEL = {
    short_break: '☕ Short break',
    long_break: '🌿 Long break'
  }

  function focusText(mins) {
    const h = Math.floor(mins / 60)
    const m = mins % 60
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
  }

  function timeOf(iso) {
    if (!iso) return ''
    const d = new Date(iso)
    if (isNaN(d)) return ''
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
</script>

<div class="stats">
  <h3 class="heading">Today</h3>

  <div class="grid">
    <div class="stat card">
      <div class="value">{stats.work_sessions}</div>
      <div class="label">{$mascot} Sessions</div>
    </div>
    <div class="stat card">
      <div class="value">{focusText(stats.total_focus_minutes)}</div>
      <div class="label">Focus time</div>
    </div>
  </div>

  <div class="stat card wide">
    <div class="value small">{stats.sessions_completed}</div>
    <div class="label">Total intervals completed today</div>
  </div>

  {#if task}
    <h3 class="heading">History · {task.title}</h3>
    <div class="history">
      {#if sessions.length === 0}
        <p class="empty">No sessions logged yet.</p>
      {/if}
      {#each sessions as s}
        <div class="row card">
          <span class="type">{s.type === 'work' ? `${$mascot} Work` : TYPE_LABEL[s.type] || s.type}</span>
          <span class="dur">{s.duration}m</span>
          <span class="when">{timeOf(s.started_at)}</span>
          <span class="done">{s.completed ? '✓' : '⋯'}</span>
        </div>
      {/each}
    </div>
  {:else}
    <div class="hint card">
      <span class="emoji">📈</span>
      <p>Select a task to see its Pomodoro history.</p>
    </div>
  {/if}
</div>

<style>
  .stats {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    height: 100%;
    overflow-y: auto;
  }
  .heading {
    margin: 0.5rem 0 0.1rem;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-muted);
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.6rem;
  }
  .stat {
    padding: 1rem;
    text-align: center;
  }
  .stat.wide {
    text-align: left;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .value {
    font-size: 1.9rem;
    font-weight: 800;
    color: var(--accent-ink);
    line-height: 1;
  }
  .value.small {
    font-size: 1.4rem;
  }
  .label {
    font-size: 0.72rem;
    color: var(--text-muted);
    margin-top: 0.35rem;
  }

  .history {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .history .row {
    display: grid;
    grid-template-columns: 1fr auto auto auto;
    align-items: center;
    gap: 0.6rem;
    padding: 0.55rem 0.75rem;
    font-size: 0.78rem;
  }
  .history .dur {
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }
  .history .when {
    color: var(--text-faint);
    font-variant-numeric: tabular-nums;
  }
  .history .done {
    color: var(--success);
  }
  .empty {
    color: var(--text-faint);
    font-size: 0.8rem;
    padding: 0.5rem;
  }

  .hint {
    padding: 1.5rem 1rem;
    text-align: center;
    color: var(--text-muted);
  }
  .hint .emoji {
    font-size: 1.6rem;
  }
  .hint p {
    margin: 0.5rem 0 0;
    font-size: 0.82rem;
  }
</style>
