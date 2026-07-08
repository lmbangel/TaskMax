<script>
  import { onMount, onDestroy } from 'svelte'
  import { timer } from '../stores/timer.js'
  import { mascot } from '../stores/appearance.js'
  import { StartPomodoro, StopPomodoro } from '../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

  export let task = null // currently selected task (or null)
  export let config = null // app config, for session durations

  // Work sessions use the accent mascot as their icon (see accents.js).
  const MODES = {
    work: { label: 'Work', icon: null, color: 'var(--accent)' },
    short_break: { label: 'Short Break', icon: '☕', color: 'var(--info)' },
    long_break: { label: 'Long Break', icon: '🌿', color: 'var(--success)' }
  }

  // Ring geometry (compact widget size).
  const R = 92
  const CIRC = 2 * Math.PI * R

  $: state = $timer
  $: mode = MODES[state.session_type] || MODES.work
  $: totalSeconds = totalFor(state.session_type)
  // When idle (nothing started yet), show the full upcoming session length.
  $: displaySeconds =
    state.is_running || state.seconds_remaining > 0 ? state.seconds_remaining : totalSeconds
  $: fraction = totalSeconds > 0 ? Math.max(0, displaySeconds / totalSeconds) : 0
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
    <span class="icon">{mode.icon || $mascot}</span>
    <span class="label">{mode.label}</span>
  </div>

  <div class="ring-wrap">
    <svg width="216" height="216" viewBox="0 0 216 216">
      <circle
        class="track"
        cx="108"
        cy="108"
        r={R}
        fill="none"
        stroke="var(--surface-2)"
        stroke-width="12"
      />
      <circle
        class="progress"
        cx="108"
        cy="108"
        r={R}
        fill="none"
        stroke={mode.color}
        stroke-width="12"
        stroke-linecap="round"
        stroke-dasharray={CIRC}
        stroke-dashoffset={dashoffset}
        transform="rotate(-90 108 108)"
      />
    </svg>
    <div class="readout">
      <div class="time">{fmt(displaySeconds)}</div>
      <div class="sub">
        {#if task}
          on <strong>{task.title}</strong>
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
    gap: 0.9rem;
    padding: 1.1rem 1rem 1.25rem;
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
    width: 216px;
    height: 216px;
  }
  .progress {
    transition: stroke-dashoffset 0.95s linear, stroke 0.3s ease;
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
    font-size: 2.5rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
    color: var(--text);
  }
  .sub {
    font-size: 0.78rem;
    color: var(--text-muted);
    max-width: 150px;
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
    min-width: 96px;
    padding: 0.6rem 0.9rem;
    font-size: 0.85rem;
  }
</style>
