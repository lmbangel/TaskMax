<script>
  import { onMount } from 'svelte'
  import { mascot } from '../stores/appearance.js'
  import { GetDailyActivity } from '../../wailsjs/go/main/App'

  export let stats = { sessions_completed: 0, work_sessions: 0, total_focus_minutes: 0 }
  export let sessions = [] // sessions for the selected task
  export let task = null

  // Work sessions use the accent mascot as their icon.
  const TYPE_LABEL = {
    short_break: '☕ Short break',
    long_break: '🌿 Long break'
  }

  // ----- Activity heatmap (last 16 weeks, GitHub-contribution style) -----
  const WEEKS = 16
  let weeks = [] // [[{date, count, level} x7] xWEEKS], columns oldest → newest
  let thisWeek = { count: 0, minutes: 0 }
  let lastWeek = { count: 0, minutes: 0 }

  function dayKey(d) {
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
  }

  function level(count) {
    if (count <= 0) return 0
    if (count === 1) return 1
    if (count <= 3) return 2
    if (count <= 5) return 3
    return 4
  }

  onMount(async () => {
    let activity = []
    try {
      activity = (await GetDailyActivity(WEEKS * 7 + 6)) || []
    } catch (e) {
      return
    }
    const byDate = Object.fromEntries(activity.map((a) => [a.date, a]))

    const today = new Date()
    today.setHours(0, 0, 0, 0)
    // Monday of the current week (getDay(): Sun=0 … Sat=6).
    const monday = new Date(today)
    monday.setDate(today.getDate() - ((today.getDay() + 6) % 7))

    const grid = []
    for (let w = WEEKS - 1; w >= 0; w--) {
      const col = []
      for (let d = 0; d < 7; d++) {
        const date = new Date(monday)
        date.setDate(monday.getDate() - w * 7 + d)
        const key = dayKey(date)
        const entry = byDate[key]
        col.push({
          date: key,
          future: date > today,
          count: entry?.count || 0,
          minutes: entry?.minutes || 0,
          level: level(entry?.count || 0)
        })
      }
      grid.push(col)
    }
    weeks = grid

    const sum = (col) =>
      col.reduce((acc, c) => ({ count: acc.count + c.count, minutes: acc.minutes + c.minutes }), {
        count: 0,
        minutes: 0
      })
    thisWeek = sum(grid[grid.length - 1])
    lastWeek = sum(grid[grid.length - 2] || [])
  })

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

  <h3 class="heading">Last {WEEKS} weeks</h3>
  <div class="card heatmap-card">
    <div class="heatmap">
      {#each weeks as col}
        <div class="hm-col">
          {#each col as cell}
            <div
              class="hm-cell l{cell.level}"
              class:future={cell.future}
              title="{cell.date}: {cell.count} session{cell.count === 1 ? '' : 's'}{cell.minutes
                ? ` · ${cell.minutes}m`
                : ''}"
            ></div>
          {/each}
        </div>
      {/each}
    </div>
    <div class="hm-footer">
      <span class="hm-week">
        This week <b>{thisWeek.count}</b>{lastWeek.count ? ` · last week ${lastWeek.count}` : ''}
      </span>
      <span class="hm-scale">
        <span class="hm-cell l0"></span>
        <span class="hm-cell l1"></span>
        <span class="hm-cell l2"></span>
        <span class="hm-cell l3"></span>
        <span class="hm-cell l4"></span>
      </span>
    </div>
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

  /* ----- Heatmap ----- */
  .heatmap-card {
    padding: 0.75rem;
  }
  .heatmap {
    display: flex;
    gap: 3px;
    justify-content: center;
  }
  .hm-col {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .hm-cell {
    width: 11px;
    height: 11px;
    border-radius: 3px;
    background: var(--surface-2);
  }
  .hm-cell.l1 {
    background: color-mix(in srgb, var(--accent) 30%, var(--surface-2));
  }
  .hm-cell.l2 {
    background: color-mix(in srgb, var(--accent) 55%, var(--surface-2));
  }
  .hm-cell.l3 {
    background: color-mix(in srgb, var(--accent) 80%, var(--surface-2));
  }
  .hm-cell.l4 {
    background: var(--accent);
  }
  .hm-cell.future {
    opacity: 0.25;
  }
  .hm-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 0.6rem;
  }
  .hm-week {
    font-size: 0.7rem;
    color: var(--text-muted);
  }
  .hm-week b {
    color: var(--accent-ink);
  }
  .hm-scale {
    display: flex;
    gap: 3px;
  }
  .hm-scale .hm-cell {
    width: 9px;
    height: 9px;
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
