<script>
  import { app, saveConfig, formatDate } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const m = $derived(st?.moteur ?? { etat: 'arrete' })
  const configure = $derived(st?.configure ?? false)

  let editing = $state(false)
  let showJeton = $state(false)
  let fields = $state({})
  let form = $state({ domaine: '', jeton: '', email: '' })

  const showForm = $derived(!configure || editing)

  function startEdit() {
    form = { domaine: st?.domaine ?? '', jeton: '', email: st?.email ?? '' }
    fields = {}
    editing = true
  }

  async function submit(event) {
    event.preventDefault()
    const res = await saveConfig({ ...form })
    fields = res?.fields ?? {}
    if (!res?.error) {
      editing = false
      showJeton = false
      form.jeton = ''
    }
  }
</script>

<div class="page">
  <div class="page-head">
    <h1>HTTPS</h1>
    <span class="pill" class:on={m.etat === 'actif'}>{m.etat === 'actif' ? 'Certificat actif' : 'Hors ligne'}</span>
  </div>
  <p class="page-intro">
    Aria obtient et renouvelle seule un vrai certificat Let's Encrypt pour ton nom de domaine. L'appli Bitwarden pour
    Android refuse les certificats faits maison : c'est ce qui la rend compatible.
  </p>
  <p class="page-intro">
    Pas de nom de domaine ? <strong>DuckDNS</strong> t'en offre un gratuitement en deux minutes (exemple.duckdns.org) :
    c'est le fournisseur que gère Aria aujourd'hui.
  </p>

  <aside class="callout" aria-label="Adresse IP fixe">
    <span class="callout-title">Garde la même adresse IP</span>
    <span>
      Ton domaine pointe vers l'adresse locale de ce PC. Si elle change, ton téléphone ne trouve plus le coffre : réserve-la
      pour ce PC dans l'interface de ta box (section DHCP, « bail statique » ou « adresse réservée »).
    </span>
  </aside>

  <div class="grid-2">
    <section class="card">
      <div class="card-head">
        <h2>Configuration</h2>
        {#if configure && !editing}<span class="pill on">Enregistrée</span>{/if}
      </div>

      {#if showForm}
        <form class="form" onsubmit={submit}>
          <label class="field">
            <span class="field-label">Sous-domaine DuckDNS</span>
            <input bind:value={form.domaine} placeholder="Ex. vault.exemple.duckdns.org" autocomplete="off" spellcheck="false" />
            {#if fields.domaine}<span class="field-error">{fields.domaine}</span>{/if}
          </label>

          <label class="field">
            <span class="field-label">Jeton DuckDNS</span>
            <span class="with-btn">
              <input
                type={showJeton ? 'text' : 'password'}
                bind:value={form.jeton}
                placeholder={st?.jetonEnregistre ? '•••••••• (laisse vide pour garder l\'actuel)' : 'Jeton affiché sur duckdns.org'}
                autocomplete="off"
                spellcheck="false"
              />
              <button type="button" class="btn-ghost small" onclick={() => (showJeton = !showJeton)} aria-label={showJeton ? 'Masquer le jeton' : 'Afficher le jeton'}>
                {showJeton ? 'Masquer' : 'Afficher'}
              </button>
            </span>
            {#if fields.jeton}<span class="field-error">{fields.jeton}</span>{/if}
          </label>

          <label class="field">
            <span class="field-label">E-mail <span class="optional">(facultatif)</span></span>
            <input bind:value={form.email} placeholder="Pour les avis de Let's Encrypt" autocomplete="off" spellcheck="false" />
            {#if fields.email}<span class="field-error">{fields.email}</span>{/if}
          </label>

          <p class="hint">Le jeton est chiffré avec DPAPI : seul ton compte Windows peut le relire.</p>

          {#if app.error && !Object.keys(fields).length}<p class="error" role="alert">{app.error}</p>{/if}

          <div class="actions">
            <button type="submit" class="btn-primary grow" disabled={app.busy}>
              {app.busy ? 'Enregistrement…' : m.etat === 'arrete' ? 'Enregistrer' : 'Enregistrer et relancer'}
            </button>
            {#if editing}
              <button type="button" class="btn-ghost" onclick={() => (editing = false)} disabled={app.busy}>Annuler</button>
            {/if}
          </div>
        </form>
      {:else}
        <div class="row"><span class="row-name">Domaine</span><span class="value mono">{st.domaine}</span></div>
        <div class="row"><span class="row-name">Jeton DuckDNS</span><span class="value mono">•••••••• chiffré</span></div>
        <div class="row"><span class="row-name">E-mail</span><span class="value">{st.email || '—'}</span></div>
        <div class="actions">
          <button class="btn-ghost grow" onclick={startEdit} disabled={app.busy}>Modifier</button>
        </div>
      {/if}
    </section>

    <section class="card">
      <h2>Certificat</h2>
      <div class="row"><span class="row-name">État</span><span class="chip" class:ok={m.etat === 'actif'}>{m.etat === 'actif' ? 'Servi sur le port 443' : m.etat === 'demarrage' ? 'En cours' : 'Inactif'}</span></div>
      <div class="row"><span class="row-name">Valable jusqu'au</span><span class="value">{formatDate(m.certificat) || '—'}</span></div>
      <div class="row"><span class="row-name">Renouvellement</span><span class="value">auto, 30 j avant</span></div>

      <span class="eyebrow">Comment ça marche</span>
      <ol class="how">
        <li>Ton domaine pointe vers l'adresse <strong>privée</strong> de ce PC (192.168.x.x), qui doit rester fixe.</li>
        <li>Aria prouve à Let's Encrypt qu'il contrôle le domaine par un enregistrement DNS (DNS-01) : aucun port à ouvrir sur la box.</li>
        <li>Hors de chez toi, ton téléphone passe par ton VPN.</li>
      </ol>
    </section>
  </div>
</div>

<style>
  .page-intro strong {
    color: var(--text);
    font-weight: 500;
  }
  .page-intro + .page-intro {
    margin-top: -12px;
  }
  .callout {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-width: 760px;
    padding: 12px 14px;
    border-left: 3px solid var(--action);
    border-radius: 0 10px 0 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--action) 8%, transparent);
    font-size: 13px;
    line-height: 1.6;
    color: var(--muted);
  }
  .callout-title {
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--action);
  }
  .optional {
    color: var(--faint);
    font-weight: 400;
  }
  .how {
    margin: 0;
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12px;
    line-height: 1.6;
    color: var(--muted);
  }
  .how strong {
    color: var(--text);
    font-weight: 500;
  }
</style>
