<script>
  import { onMount } from 'svelte'
  import Switch from '../lib/Switch.svelte'
  import {
    app, loadBackups, backupNow, restoreBackup, importVault, chooseBackupFolder, openBackupFolder,
    setDailyBackup, exportSettings, importSettings, formatDateHeure, formatSince, formatTaille,
  } from '../lib/state.svelte.js'

  const b = $derived(app.backups)
  const liste = $derived(b?.liste ?? [])
  const enLigne = $derived(['actif', 'demarrage'].includes(app.status?.moteur?.etat))

  let aConfirmer = $state('') // sauvegarde en attente de confirmation de restauration
  let message = $state('')

  let phraseExport = $state('')
  let phraseExport2 = $state('')
  let phraseImport = $state('')
  let montrerPhrases = $state(false)
  const phraseOk = $derived(phraseExport.length >= 8 && phraseExport === phraseExport2)

  onMount(loadBackups)

  async function sauvegarder() {
    message = ''
    const info = await backupNow()
    if (info?.nom) message = `Sauvegarde créée : ${info.nom} (${formatTaille(info.taille)})`
  }

  async function restaurer(nom) {
    if (aConfirmer !== nom) {
      aConfirmer = nom
      return
    }
    aConfirmer = ''
    message = ''
    await restoreBackup(nom)
    if (!app.error) message = 'Coffre restauré. L\'ancien a été mis de côté à côté du dossier data, rien n\'est supprimé.'
  }

  async function importer() {
    message = ''
    const ok = await importVault()
    if (ok && !app.error) message = 'Ancien coffre importé. Le précédent a été mis de côté, rien n\'est supprimé.'
  }

  async function exporter() {
    message = ''
    const ok = await exportSettings(phraseExport)
    if (ok && !app.error) {
      message = 'Réglages exportés. Garde ta phrase de passe en lieu sûr : sans elle, le jeton sera à ressaisir.'
      phraseExport = phraseExport2 = ''
    }
  }

  async function importerReglages() {
    message = ''
    const res = await importSettings(phraseImport)
    if (res && !res.annule && !res.error) {
      message = res.jeton
        ? `Réglages importés pour ${res.domaine}, jeton compris.`
        : `Réglages importés pour ${res.domaine}. Recolle ton jeton DuckDNS sur la page HTTPS.`
      phraseImport = ''
    }
  }
</script>

<div class="page">
  <div class="page-head">
    <h1>Sauvegardes</h1>
    <span class="pill" class:on={liste.length > 0}>
      {liste.length ? `Dernière : ${formatSince(liste[0].date).replace('depuis', 'il y a')}` : 'Aucune sauvegarde'}
    </span>
  </div>
  <p class="page-intro">
    Tout ton coffre tient dans un petit zip : base, clés du serveur, pièces jointes et Sends. Tes mots de passe y restent
    chiffrés par ton mot de passe maître.
  </p>

  {#if message}<p class="banner ok" role="status">{message}</p>{/if}
  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}

  <div class="grid-2">
    <section class="card">
      <h2>Coffre</h2>
      <button class="btn-primary big" onclick={sauvegarder} disabled={app.busy || !app.status?.configure}>
        {app.busy ? 'En cours…' : 'Sauvegarder maintenant'}
      </button>
      <p class="hint">
        {enLigne ? 'Le serveur se met en pause quelques secondes, ton téléphone réessaie tout seul.' : 'Aussi possible d\'un clic droit sur l\'icône Aria.'}
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Sauvegarde quotidienne</span>
          <span class="row-detail">Une par jour tant qu'Aria tourne. Les {b?.garder ?? 10} plus récentes sont gardées.</span>
        </div>
        <Switch checked={b?.auto ?? true} disabled={app.busy} label="Sauvegarde quotidienne" onchange={setDailyBackup} />
      </div>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Dossier</span>
          <span class="path">{b?.dossier ?? ''}</span>
        </div>
        <span class="btns">
          <button class="btn-ghost small" onclick={chooseBackupFolder} disabled={app.busy}>Changer</button>
          <button class="btn-ghost small" onclick={openBackupFolder} disabled={app.busy}>Ouvrir</button>
        </span>
      </div>
      {#if b?.parDefaut}
        <p class="hint warn-text">
          Ce dossier est sur le même disque que le coffre : si le disque lâche, tu perds les deux. Choisis plutôt un dossier
          OneDrive ou un disque externe.
        </p>
      {/if}
    </section>

    <section class="card">
      <h2>Historique</h2>
      {#if liste.length}
        <div class="list">
          {#each liste as s (s.nom)}
            <div class="row">
              <div class="row-text">
                <span class="row-name">{formatDateHeure(s.date)}</span>
                <span class="row-detail mono">{formatTaille(s.taille)}</span>
              </div>
              <button
                class="small {aConfirmer === s.nom ? 'btn-primary' : 'btn-ghost'}"
                onclick={() => restaurer(s.nom)}
                onblur={() => aConfirmer === s.nom && (aConfirmer = '')}
                disabled={app.busy}
              >
                {aConfirmer === s.nom ? 'Confirmer ?' : 'Restaurer'}
              </button>
            </div>
          {/each}
        </div>
        <p class="hint">Restaurer remplace le coffre actuel, qui est d'abord mis de côté : rien n'est jamais supprimé.</p>
      {:else}
        <p class="card-text">Aucune sauvegarde pour l'instant.</p>
      {/if}
      <div class="actions">
        <button class="btn-ghost grow" onclick={importer} disabled={app.busy}>Importer un ancien coffre…</button>
      </div>
      <p class="hint">Un ancien dossier data, comme celui de C:\Vaultwarden.</p>
    </section>
  </div>

  <section class="card">
    <div class="card-head">
      <h2>Réglages d'Aria</h2>
      <button class="btn-ghost small" onclick={() => (montrerPhrases = !montrerPhrases)}>
        {montrerPhrases ? 'Masquer les phrases' : 'Afficher les phrases'}
      </button>
    </div>
    <p class="card-text">
      Domaine, e-mail et jeton DuckDNS, dans un fichier à part du coffre. Le jeton y est chiffré par ta phrase de passe :
      sur un autre PC, tu retrouves tout sans rien ressaisir.
    </p>
    <div class="grid-2">
      <form class="form" onsubmit={(e) => { e.preventDefault(); exporter() }}>
        <span class="eyebrow">Exporter</span>
        <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseExport} placeholder="Phrase de passe (8 caractères min.)" autocomplete="new-password" />
        <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseExport2} placeholder="La même, pour confirmer" autocomplete="new-password" />
        {#if phraseExport2 && phraseExport !== phraseExport2}<span class="field-error">Les deux phrases sont différentes.</span>{/if}
        <button type="submit" class="btn-ghost" disabled={app.busy || !phraseOk || !app.status?.configure}>Exporter les réglages…</button>
      </form>
      <form class="form" onsubmit={(e) => { e.preventDefault(); importerReglages() }}>
        <span class="eyebrow">Importer</span>
        <input type={montrerPhrases ? 'text' : 'password'} bind:value={phraseImport} placeholder="Phrase de passe du fichier" autocomplete="off" />
        <p class="hint">Sans phrase : domaine et e-mail seulement, le jeton sera à recoller.</p>
        <button type="submit" class="btn-ghost" disabled={app.busy}>Importer des réglages…</button>
      </form>
    </div>
  </section>
</div>

<style>
  .big {
    min-height: 52px;
    font-size: 15px;
  }
  .btns {
    display: flex;
    gap: 8px;
    flex: none;
  }
  .list {
    display: flex;
    flex-direction: column;
    max-height: 300px;
    overflow-y: auto;
    padding-right: 4px;
  }
  .warn-text {
    color: var(--warn);
  }
  .banner.ok {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    border-color: color-mix(in srgb, var(--accent) 40%, transparent);
    color: #bdf9ff;
  }
</style>
