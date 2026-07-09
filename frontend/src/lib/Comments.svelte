<script>
  import { onMount, onDestroy } from 'svelte'
  import { GetComments, AddComment } from '../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

  export let taskId = 0

  let comments = []
  let draft = ''
  let open = false
  let busy = false

  // Reload whenever the selected task changes (and on first render).
  $: if (taskId) load(taskId)

  async function load(id) {
    try {
      comments = (await GetComments(id)) || []
    } catch (e) {
      comments = []
    }
  }

  async function add() {
    const body = draft.trim()
    if (!body || busy) return
    busy = true
    try {
      await AddComment(taskId, body)
      draft = ''
      await load(taskId)
    } finally {
      busy = false
    }
  }

  function who(c) {
    if (c.author) return c.source === 'agent' ? `🤖 ${c.author}` : c.author
    return c.source === 'agent' ? '🤖 Agent' : 'You'
  }

  function when(c) {
    const d = new Date(c.CreatedAt)
    return (
      d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) +
      ' ' +
      d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
    )
  }

  onMount(() => {
    // An agent commented over MCP — refresh the open trail live.
    EventsOn('comments:changed', (id) => {
      if (id === taskId) load(taskId)
    })
  })
  onDestroy(() => EventsOff('comments:changed'))
</script>

<div class="comments">
  <button class="toggle" on:click={() => (open = !open)}>
    💬 Comments{comments.length ? ` (${comments.length})` : ''}
    <span class="chev">{open ? '▾' : '▸'}</span>
  </button>

  {#if open}
    {#if comments.length}
      <ul class="list">
        {#each comments as c (c.ID)}
          <li class="comment" class:agent={c.source === 'agent'}>
            <div class="meta">
              <span class="who">{who(c)}</span>
              <span class="ts">{when(c)}</span>
            </div>
            <p class="body">{c.body}</p>
          </li>
        {/each}
      </ul>
    {/if}

    <form class="add" on:submit|preventDefault={add}>
      <input placeholder="Add a comment…" bind:value={draft} disabled={busy} />
      <button class="btn btn-accent" type="submit" disabled={busy || !draft.trim()}>↑</button>
    </form>
  {/if}
</div>

<style>
  .comments {
    margin-top: 0.6rem;
    border-top: 1px solid var(--surface-2);
    padding-top: 0.5rem;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    width: 100%;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    padding: 0.15rem 0;
    text-align: left;
  }
  .toggle:hover {
    color: var(--text);
  }
  .chev {
    margin-left: auto;
    font-size: 0.65rem;
  }
  .list {
    list-style: none;
    margin: 0.45rem 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    max-height: 160px;
    overflow-y: auto;
  }
  .comment {
    background: var(--surface-2);
    border-radius: var(--radius-input);
    padding: 0.4rem 0.55rem;
  }
  .comment.agent {
    background: var(--accent-soft);
  }
  .meta {
    display: flex;
    align-items: center;
    gap: 0.45rem;
  }
  .who {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--accent-ink);
  }
  .comment:not(.agent) .who {
    color: var(--text-muted);
  }
  .ts {
    font-size: 0.62rem;
    color: var(--text-faint);
  }
  .body {
    margin: 0.25rem 0 0;
    font-size: 0.76rem;
    line-height: 1.45;
    color: var(--text);
    white-space: pre-wrap;
    word-break: break-word;
  }
  .add {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.5rem;
  }
  .add input {
    flex: 1 1 auto;
    font-size: 0.76rem;
    padding: 0.4rem 0.6rem;
  }
  .add .btn {
    flex: 0 0 auto;
    padding: 0.35rem 0.65rem;
    font-size: 0.8rem;
  }
</style>
