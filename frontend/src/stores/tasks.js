import { writable } from 'svelte/store'
import {
  GetAllTasks,
  CreateTask,
  UpdateTask,
  DeleteTask,
  ReorderTasks
} from '../../wailsjs/go/main/App'

/**
 * Task store. Holds the full task list and exposes CRUD helpers that keep the
 * store in sync with the Go backend.
 */
function createTaskStore() {
  const { subscribe, set } = writable([])

  async function refresh() {
    const list = await GetAllTasks()
    set(list || [])
  }

  async function create(task) {
    const created = await CreateTask(task)
    await refresh()
    return created
  }

  async function updateTask(task) {
    const saved = await UpdateTask(task)
    await refresh()
    return saved
  }

  async function remove(id) {
    await DeleteTask(id)
    await refresh()
  }

  // Optimistically apply a new order locally, then persist it.
  async function reorder(orderedTasks) {
    set(orderedTasks)
    await ReorderTasks(orderedTasks.map((t) => t.ID))
    await refresh()
  }

  return {
    subscribe,
    set,
    refresh,
    create,
    update: updateTask,
    remove,
    reorder
  }
}

export const tasks = createTaskStore()
