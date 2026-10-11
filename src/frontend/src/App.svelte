<script>
  import { onMount } from 'svelte'
  import Icon from './lib/Icon.svelte'
  import Logo from './lib/Logo.svelte'
  import { app, refresh, refreshJournal, startPolling, openVault } from './lib/state.svelte.js'
  import Home from './pages/Home.svelte'
  import Https from './pages/Https.svelte'
  import Reglages from './pages/Reglages.svelte'
  import Journal from './pages/Journal.svelte'

  const nav = [
    { id: 'home', label: 'Accueil', cmd: 'accueil', icon: 'home' },
    { id: 'https', label: 'HTTPS', cmd: 'https', icon: 'lock' },
    { id: 'reglages', label: 'Réglages', cmd: 'réglages', icon: 'settings' },
    { id: 'journal', label: 'Journal', cmd: 'journal', icon: 'journal' },
  ]

  const etat = $derived(app.status?.moteur?.etat ?? 'arrete')
  const version = $derived(app.status?.version ?? '')

  onMount(() => {
    refresh()
    refreshJournal()
    startPolling()
  })
</script>

<div class="shell">
  <nav class="sidebar" aria-label="Sections">
    <div class="brand">
      <Logo size={38} dim={etat !== 'actif'} />
      <span class="brand-name label"><span class="brand-top">ARIA</span><span class="brand-sub">VAULT</span></span>
    </div>

    <ul class="nav">
      {#each nav as item (item.id)}
        {@const lit = (item.id === 'home' && etat === 'actif') || (item.id === 'journal' && etat === 'erreur')}
        <li>
          <button
            class="nav-item"
            class:active={app.page === item.id}
            aria-current={app.page === item.id ? 'page' : undefined}
            aria-label={item.label}
            title={item.label}
            onclick={() => {
              app.page = item.id
              if (item.id === 'journal') refreshJournal()
            }}
          >
            <span class="nav-icon">
              <Icon name={item.icon} />
              {#if lit}<span class="dot icon-dot" class:alert={etat === 'erreur'} aria-hidden="true"></span>{/if}
            </span>
            <span class="prompt label" aria-hidden="true">&gt;</span>
            <span class="label">{item.cmd}</span>
            {#if lit}<span class="dot label" class:alert={etat === 'erreur'} aria-hidden="true"></span>{/if}
          </button>
        </li>
      {/each}
    </ul>

    <div class="sidebar-foot">
      <button
        class="open"
        disabled={!app.status?.domaine || app.busy}
        aria-label="Ouvrir le coffre dans le navigateur"
        title={app.status?.domaine ? `https://${app.status.domaine}` : 'Configure d\'abord HTTPS'}
        onclick={openVault}
      >
        <span class="nav-icon"><Icon name="external" size={18} /></span>
        <span class="label">ouvrir --coffre</span>
      </button>
      <span class="version label">aria-vaultwarden {version ? `v${version}` : ''}</span>
    </div>
  </nav>

  <main class="content">
    {#if app.loadError}
      <p class="banner" role="alert">Impossible de lire l'état d'Aria : {app.loadError}</p>
    {/if}
    {#if app.status?.erreur}
      <p class="banner" role="alert">{app.status.erreur}</p>
    {/if}

    {#if app.page === 'home'}
      <Home />
    {:else if app.page === 'https'}
      <Https />
    {:else if app.page === 'reglages'}
      <Reglages />
    {:else if app.page === 'journal'}
      <Journal />
    {/if}
  </main>
</div>

<style>
  .shell {
    display: grid;
    grid-template-columns: 216px minmax(0, 1fr);
    height: 100vh;
  }

  .sidebar {
    background: var(--sidebar);
    border-right: 1px solid var(--line-soft);
    padding: 18px 12px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    min-width: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 4px;
    min-height: 40px;
  }
  .brand-name {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }
  .brand-top {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 17px;
    color: #ffffff;
    letter-spacing: 0.5px;
  }
  .brand-sub {
    font-size: 11px;
    letter-spacing: 3px;
    color: var(--accent);
  }

  .nav {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .nav-item {
    width: 100%;
    min-height: 42px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    gap: 8px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 500;
    text-align: left;
  }
  .nav-item:hover:not(:disabled) {
    background: var(--hover);
    color: var(--text);
  }
  .nav-item.active {
    background: #0b1a30;
    color: var(--accent);
  }
  .prompt {
    color: var(--dim);
  }
  .nav-item.active .prompt {
    color: var(--action);
  }
  /* En grand, la console suffit : les icônes servent quand la barre est repliée */
  .nav-icon {
    position: relative;
    display: none;
    flex: none;
  }
  .dot {
    flex: none;
    margin-left: auto;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 8px var(--accent);
  }
  .dot.alert {
    background: var(--danger);
    box-shadow: 0 0 8px var(--danger);
  }
  .icon-dot {
    position: absolute;
    right: -3px;
    top: -3px;
    margin: 0;
    border: 2px solid var(--sidebar);
    width: 9px;
    height: 9px;
  }
  .label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sidebar-foot {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .open {
    min-height: 40px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
  }
  .open:hover:not(:disabled) {
    border-color: var(--action);
    color: var(--action);
  }
  .open:disabled {
    opacity: 0.45;
  }
  .version {
    padding: 0 4px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--faint);
  }

  .content {
    overflow-y: auto;
    padding: 26px 30px 30px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Fenêtre étroite : la barre latérale se replie en icônes */
  @media (max-width: 760px) {
    .shell {
      grid-template-columns: 68px minmax(0, 1fr);
    }
    .sidebar {
      padding: 16px 8px;
      align-items: center;
    }
    .brand {
      padding: 0;
      justify-content: center;
    }
    .label {
      display: none;
    }
    .nav-icon {
      display: flex;
    }
    .nav-item,
    .open {
      justify-content: center;
      padding: 0;
      width: 48px;
      min-height: 48px;
    }
    .nav {
      align-items: center;
    }
    .content {
      padding: 20px;
    }
  }
</style>
