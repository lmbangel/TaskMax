<script>
  import { createEventDispatcher } from 'svelte'
  import { flip } from 'svelte/animate'

  export let tasks = []
  export let selectedId = null

  const dispatch = createEventDispatcher()

  const STATUS_CYCLE = { todo: 'in_progress', in_progress: 'done', done: 'todo' }
  const STATUS_META = {
    todo: { label: 'To Do', dot: 'var(--text-faint)' },
    in_progress: { label: 'In Progress', dot: 'var(--info)' },
    done: { label: 'Done', dot: 'var(--success)' }
  }

  let dragIndex = null
  let overIndex = null

  function tagList(tags) {
    return (tags || '')
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
  }

  function cycleStatus(task, e) {
    e.stopPropagation()
    dispatch('statusChange', { ...task, Status: STATUS_CYCLE[task.Status] || 'todo' })
  }

  function onDragStart(i) {
    dragIndex = i
  }
  function onDragOver(i, e) {
    e.preventDefault()
    overIndex = i
  }
  function onDrop(i) {
    if (dragIndex === null || dragIndex === i) {
      dragIndex = null
      overIndex = null
      return
    }
    const next = [...tasks]
    const [moved] = next.splice(dragIndex, 1)
    next.splice(i, 0, moved)
    dragIndex = null
    overIndex = null
    dispatch('reorder', next)
  }
  function onDragEnd() {
    dragIndex = null
    overIndex = null
  }
</script>

<div class="list">
  {#if tasks.length === 0}
    <div class="empty">
      <span class="emoji">🌱</span>
      <p>No tasks here yet.</p>
      <small>Create one to get started.</small>
    </div>
  {/if}

  {#each tasks as task, i (task.ID)}
    <div
      class="task"
      class:selected={task.ID === selectedId}
      class:done={task.Status === 'done'}
      class:dragover={overIndex === i && dragIndex !== null}
      draggable="true"
      animate:flip={{ duration: 220 }}
      on:click={() => dispatch('select', task)}
      on:dragstart={() => onDragStart(i)}
      on:dragover={(e) => onDragOver(i, e)}
      on:drop={() => onDrop(i)}
      on:dragend={onDragEnd}
      role="button"
      tabindex="0"
      on:keydown={(e) => e.key === 'Enter' && dispatch('select', task)}
    >
      <button
        class="status"
        title={STATUS_META[task.Status]?.label}
        style="--dot: {STATUS_META[task.Status]?.dot}"
        on:click={(e) => cycleStatus(task, e)}
      >
        {#if task.Status === 'done'}✓{/if}
      </button>

      <div class="body">
        <div class="top">
          <span class="title">{task.Title}</span>
          <span class="badge {task.Priority}">{task.Priority}</span>
        </div>

        {#if tagList(task.Tags).length || task.PomodoroCount > 0}
          <div class="meta">
            {#each tagList(task.Tags) as tag}
              <span class="chip">#{tag}</span>
            {/each}
            {#if task.PomodoroCount > 0}
              <span class="poms">🍅 {task.PomodoroCount}</span>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  {/each}
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.55rem;
    overflow-y: auto;
    padding: 0.25rem 0.1rem;
  }

  .task {
    display: flex;
    align-items: flex-start;
    gap: 0.7rem;
    padding: 0.75rem 0.85rem;
    background: var(--surface);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow);
    cursor: pointer;
    transition: transform 0.08s ease, background 0.15s ease, box-shadow 0.15s ease;
    border-left: 3px solid transparent;
  }
  .task:hover {
    background: var(--surface-hover);
  }
  .task.selected {
    border-left-color: var(--accent);
    background: var(--surface-hover);
  }
  .task.done .title {
    text-decoration: line-through;
    color: var(--text-faint);
  }
  .task.dragover {
    box-shadow: 0 0 0 2px var(--accent);
  }

  .status {
    flex: 0 0 auto;
    width: 22px;
    height: 22px;
    margin-top: 1px;
    border-radius: 7px;
    background: var(--surface-2);
    box-shadow: inset 0 0 0 2px var(--dot);
    color: var(--success);
    font-size: 0.8rem;
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
  }
  .status:hover {
    filter: brightness(1.15);
  }

  .body {
    flex: 1 1 auto;
    min-width: 0;
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .title {
    font-weight: 600;
    font-size: 0.9rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    margin-top: 0.45rem;
  }
  .poms {
    font-size: 0.7rem;
    color: var(--text-muted);
    font-weight: 600;
  }

  .empty {
    text-align: center;
    color: var(--text-muted);
    padding: 2.5rem 1rem;
  }
  .empty .emoji {
    font-size: 2rem;
  }
  .empty p {
    margin: 0.6rem 0 0.2rem;
    font-weight: 600;
    color: var(--text);
  }
  .empty small {
    color: var(--text-faint);
  }
</style>
