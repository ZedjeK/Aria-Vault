<script>
  import Switch from '../lib/Switch.svelte'
  import { app, start, stop, setAutostart, setInscriptions, openData } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const etat = $derived(st?.moteur?.etat ?? 'arrete')
  const running = $derived(etat === 'actif' || etat === 'demarrage')
  const label = { actif: 'En ligne', demarrage: 'Démarrage…', erreur: 'Erreur', arrete: 'Arrêté' }
</script>

<div class="page">
  <div class="page-head">
    <h1>Réglages</h1>
  </div>

  <div class="grid-2">
    <section class="card">
      <h2>Coffre</h2>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Serveur</span>
          <span class="row-detail">
            <span class="chip" class:ok={etat === 'actif'} class:warn={etat === 'demarrage'} class:fail={etat === 'erreur'}>{label[etat]}</span>
          </span>
        </div>
        {#if running}
          <button class="btn-ghost small" onclick={stop} disabled={app.busy}>{app.busy ? 'Arrêt…' : 'Arrêter'}</button>
        {:else}
          <button class="btn-primary small" onclick={start} disabled={app.busy || !st?.configure}>
            {app.busy ? 'Démarrage…' : etat === 'erreur' ? 'Relancer' : 'Démarrer'}
          </button>
        {/if}
      </div>
      <p class="hint">
        Aussi possible d'un clic droit sur l'icône Aria, dans la zone de notification. Fermer la fenêtre n'arrête pas le
        serveur.
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Dossier du coffre</span>
          <span class="path">{st?.donnees ?? ''}</span>
        </div>
        <button class="btn-ghost small" onclick={openData} disabled={app.busy}>Ouvrir</button>
      </div>
      <p class="hint">
        Base, clés du serveur et pièces jointes : tout est là. Pour une sauvegarde, arrête le serveur (bouton ci-dessus)
        avant de copier ce dossier.
      </p>

      <div class="row">
        <div class="row-text">
          <span class="row-name">Autoriser les inscriptions</span>
          <span class="row-detail">À n'ouvrir que le temps de créer un compte.</span>
        </div>
        <Switch checked={st?.inscriptions ?? false} disabled={app.busy} label="Autoriser les inscriptions" onchange={setInscriptions} />
      </div>
      {#if running}
        <p class="hint">Changer ce réglage relance le serveur quelques secondes.</p>
      {/if}
    </section>

    <section class="card">
      <h2>Démarrage</h2>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Lancer à l'ouverture de session</span>
          <span class="row-detail">Aria démarre en silence dans la zone de notification et sert ton coffre.</span>
        </div>
        <Switch checked={st?.autostart ?? false} disabled={app.busy} label="Lancer à l'ouverture de session" onchange={setAutostart} />
      </div>
      <p class="hint">
        Le serveur ne tourne que pendant ta session Windows. Session fermée ou PC en veille : ton téléphone garde sa copie
        hors ligne et se resynchronise au retour.
      </p>
    </section>
  </div>

  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}
</div>
