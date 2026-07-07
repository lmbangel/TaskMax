import { writable } from 'svelte/store'
import { GetTimerState } from '../../wailsjs/go/main/App'

const EMPTY = {
  seconds_remaining: 0,
  session_type: 'work',
  is_running: false,
  active_task_id: 0
}

/**
 * Timer store. Polls the Go timer state once per second. The authoritative
 * countdown lives in Go; this store simply mirrors it for the UI.
 */
function createTimerStore() {
  const { subscribe, set } = writable({ ...EMPTY })
  let interval = null

  async function poll() {
    try {
      const state = await GetTimerState()
      if (state) set(state)
    } catch (e) {
      // Runtime not ready yet (e.g. during dev reload) — ignore.
    }
  }

  function start() {
    if (interval) return
    poll()
    interval = setInterval(poll, 1000)
  }

  function stop() {
    if (interval) {
      clearInterval(interval)
      interval = null
    }
  }

  return { subscribe, set, poll, start, stop }
}

export const timer = createTimerStore()
