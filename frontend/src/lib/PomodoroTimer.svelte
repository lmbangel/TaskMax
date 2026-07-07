<script>
  import { onMount, onDestroy } from 'svelte'
  import { timer } from '../stores/timer.js'
  import { StartPomodoro, StopPomodoro } from '../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

  export let task = null // currently selected task (or null)
  export let config = null // app config, for session durations

  const MODES = {
    work: { label: 'Work', icon: '🍅', color: 'var(--accent)' },
    short_break: { label: 'Short Break', icon: '☕', color: 'var(--info)' },
    long_break: { label: 'Long Break', icon: '🌿', color: 'var(--success)' }
  }

  // Ring geometry.
  const R = 120
  const CIRC = 2 * Math.PI * R

  $: state = $timer
  $: mode = MODES[state.session_type] || MODES.work
  $: totalSeconds = totalFor(state.session_type)
  $: fraction = totalSeconds > 0 ? Math.max(0, state.seconds_remaining / totalSeconds) : 0
  $: dashoffset = CIRC * (1 - fraction)
  $: activeTaskID = task ? task.ID : state.active_task_id || 0

  function totalFor(type) {
    if (!config) return 25 * 60
    if (type === 'short_break') return config.pomodoro.short_break * 60
    if (type === 'long_break') return config.pomodoro.long_break * 60
    return config.pomodoro.work_duration * 60
  }

  function fmt(secs) {
    const s = Math.max(0, secs)
    const m = Math.floor(s / 60)
    const r = s % 60
    return `${String(m).padStart(2, '0')}:${String(r).padStart(2, '0')}`
  }

  async function start() {
    await StartPomodoro(activeTaskID, state.session_type || 'work')
    timer.poll()
  }

  async function pause() {
    await StopPomodoro()
    timer.poll()
  }

  // Skip flips work <-> break without completing the current session.
  async function skip() {
    const next = state.session_type === 'work' ? 'short_break' : 'work'
    await StopPomodoro()
    await StartPomodoro(activeTaskID, next)
    timer.poll()
  }

  // When a session completes in Go, auto-advance into the next one.
  function onComplete(payload) {
    if (payload && payload.next_type) {
      StartPomodoro(payload.task_id || 0, payload.next_type).then(() => timer.poll())
    }
  }

  onMount(() => {
    EventsOn('pomodoro:complete', onComplete)
    timer.start()
  })

  onDestroy(() => {
    EventsOff('pomodoro:complete')
  })
</script>

<div class="timer card">
  <div class="mode" style="--mode-color: {mode.color}">
    <span class="icon">{mode.icon}</span>
    <span class="label">{mode.label}</span>
  </div>

  <div class="ring-wrap">
    <svg width="280" height="280" viewBox="0 0 280 280">
      <circle
        class="track"
        cx="140"
        cy="140"
        r={R}
        fill="none"
        stroke="var(--surface-2)"
        stroke-width="14"
      />
      <circle
        class="progress"
        cx="140"
        cy="140"
        r={R}
        fill="none"
        stroke={mode.color}
        stroke-width="14"
        stroke-linecap="round"
        stroke-dasharray={CIRC}
        stroke-dashoffset={dashoffset}
        transform="rotate(-90 140 140)"
      />
    </svg>
    <div class="readout">
      <div class="time">{fmt(state.seconds_remaining)}</div>
      <div class="sub">
        {#if task}
          on <strong>{task.Title}</strong>
        {:else}
          {state.is_running ? 'focusing' : 'ready'}
        {/if}
      </div>
    </div>
  </div>

  <div class="controls">
    {#if state.is_running}
      <button class="btn" on:click={pause}>⏸ Pause</button>
    {:else}
      <button class="btn btn-accent" on:click={start}>
        ▶ {state.seconds_remaining > 0 && state.seconds_remaining < totalSeconds ? 'Resume' : 'Start'}
      </button>
    {/if}
    <button class="btn btn-ghost" on:click={skip}>⏭ Skip</button>
  </div>
</div>

<style>
  .timer {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
    padding: 1.75rem 2rem 2rem;
  }

  .mode {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.9rem;
    border-radius: 999px;
    background: color-mix(in srgb, var(--mode-color) 16%, transparent);
    color: var(--mode-color);
    font-weight: 700;
    font-size: 0.85rem;
    letter-spacing: 0.02em;
  }
  .mode .icon {
    font-size: 1.05rem;
  }

  .ring-wrap {
    position: relative;
    width: 280px;
    height: 280px;
  }
  .progress {
    transition: stroke-dashoffset 0.95s linear, stroke 0.3s ease;
    filter: drop-shadow(0 0 6px color-mix(in srgb, var(--mode-color) 45%, transparent));
  }

  .readout {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
  }
  .time {
    font-size: 3.4rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    color: var(--text);
  }
  .sub {
    font-size: 0.85rem;
    color: var(--text-muted);
    max-width: 200px;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sub strong {
    color: var(--text);
  }

  .controls {
    display: flex;
    gap: 0.75rem;
  }
  .controls .btn {
    min-width: 108px;
    padding: 0.7rem 1.1rem;
    font-size: 0.9rem;
  }
</style>
