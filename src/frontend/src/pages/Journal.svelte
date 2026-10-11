<script>
  import { tick } from 'svelte'
  import { app, refreshJournal } from '../lib/state.svelte.js'

  let list = $state()
  const marks = { ok: '✓', erreur: '✗', info: '›' }

  // Suit les nouvelles lignes, sauf si l'utilisateur est remonté lire.
  $effect(() => {
    app.journal.length
    if (!list) return
    const atBottom = list.scrollHeight - list.scrollTop - list.clientHeight < 40
    if (atBottom) tick().then(() => (list.scrollTop = list.scrollHeight))
  })
</script>

<div class="page">
  <div class="page-head">
    <h1>Journal</h1>
    <button class="btn-ghost small" onclick={refreshJournal}>Actualiser</button>
  </div>
  <p class="page-intro">
    Les derniers événements d'Aria, de Vaultwarden et du certificat. Copie complète dans
    <span class="mono">{app.status?.base ?? '%LOCALAPPDATA%\\AriaVaultwarden'}\aria.log</span>.
  </p>

  <section class="card">
    {#if app.journal.length}
      <ol class="log scroll" bind:this={list}>
        {#each app.journal as e, i (i)}
          <li class:fail={e.niveau === 'erreur'} class:ok={e.niveau === 'ok'}>
            <span>{marks[e.niveau] ?? '›'}</span>
            <span class="time">{e.heure}</span>
            <span>{e.texte}</span>
          </li>
        {/each}
      </ol>
    {:else}
      <p class="card-text">Rien pour l'instant.</p>
    {/if}
  </section>
</div>

<style>
  .scroll {
    max-height: calc(100vh - 230px);
    overflow-y: auto;
  }
  .log li .time {
    color: var(--faint);
    flex: none;
  }
  .log li.fail .time {
    color: var(--danger);
  }
</style>
