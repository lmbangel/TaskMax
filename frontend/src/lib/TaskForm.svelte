<script>
  import { createEventDispatcher } from 'svelte'

  export let open = false
  export let task = null // null => create mode; object => edit mode

  const dispatch = createEventDispatcher()

  let title = ''
  let description = ''
  let priority = 'medium'
  let tags = ''
  let dueDate = ''

  // Re-seed the form whenever it opens or the target task changes.
  $: if (open) seed(task)

  let lastSeeded = null
  function seed(t) {
    if (t === lastSeeded) return
    lastSeeded = t
    title = t?.Title || ''
    description = t?.Description || ''
    priority = t?.Priority || 'medium'
    tags = t?.Tags || ''
    dueDate = t?.DueDate ? String(t.DueDate).slice(0, 10) : ''
  }

  function close() {
    lastSeeded = null
    dispatch('close')
  }

  function save() {
    if (!title.trim()) return
    const payload = {
      ...(task || {}),
      Title: title.trim(),
      Description: description.trim(),
      Priority: priority,
      Tags: tags
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
        .join(','),
      DueDate: dueDate ? `${dueDate}T00:00:00Z` : null
    }
    dispatch('save', payload)
    close()
  }
</script>

{#if open}
  <div class="scrim" on:click={close} role="presentation"></div>
  <aside class="panel" class:open>
    <header>
      <h2>{task ? 'Edit Task' : 'New Task'}</h2>
      <button class="btn btn-ghost close" on:click={close}>✕</button>
    </header>

    <div class="fields">
      <div>
        <label for="tf-title">Title</label>
        <input id="tf-title" bind:value={title} placeholder="What needs doing?" />
      </div>

      <div>
        <label for="tf-desc">Description</label>
        <textarea id="tf-desc" bind:value={description} placeholder="Add some detail…"></textarea>
      </div>

      <div class="row">
        <div>
          <label for="tf-priority">Priority</label>
          <select id="tf-priority" bind:value={priority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
          </select>
        </div>
        <div>
          <label for="tf-due">Due date</label>
          <input id="tf-due" type="date" bind:value={dueDate} />
        </div>
      </div>

      <div>
        <label for="tf-tags">Tags <span class="hint">(comma separated)</span></label>
        <input id="tf-tags" bind:value={tags} placeholder="design, urgent" />
      </div>
    </div>

    <footer>
      <button class="btn btn-ghost" on:click={close}>Cancel</button>
      <button class="btn btn-accent" on:click={save} disabled={!title.trim()}>
        {task ? 'Save changes' : 'Create task'}
      </button>
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
    width: min(420px, 92vw);
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
    padding: 0.75rem 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  .row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.9rem;
  }
  .hint {
    color: var(--text-faint);
    font-weight: 400;
    text-transform: none;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.6rem;
    padding: 1rem 1.5rem 1.25rem;
    border-top: 1px solid var(--surface-2);
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
