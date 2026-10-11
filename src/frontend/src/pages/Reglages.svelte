<script>
  import Switch from '../lib/Switch.svelte'
  import { app, setAutostart, setInscriptions, openData } from '../lib/state.svelte.js'

  const st = $derived(app.status)
  const running = $derived(['actif', 'demarrage'].includes(st?.moteur?.etat))
</script>

<div class="page">
  <div class="page-head">
    <h1>Réglages</h1>
  </div>

  <div class="grid-2">
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
        Le coffre ne tourne que pendant ta session Windows. Session fermée ou PC en veille : ton téléphone garde sa copie
        hors ligne et se resynchronise au retour.
      </p>
    </section>

    <section class="card">
      <h2>Coffre</h2>
      <div class="row">
        <div class="row-text">
          <span class="row-name">Autoriser les inscriptions</span>
          <span class="row-detail">À n'ouvrir que le temps de créer un compte.</span>
        </div>
        <Switch checked={st?.inscriptions ?? false} disabled={app.busy} label="Autoriser les inscriptions" onchange={setInscriptions} />
      </div>
      {#if running}
        <p class="hint">Changer ce réglage relance le coffre quelques secondes.</p>
      {/if}
      <div class="row">
        <div class="row-text">
          <span class="row-name">Dossier du coffre</span>
          <span class="path">{st?.donnees ?? ''}</span>
        </div>
        <button class="btn-ghost small" onclick={openData} disabled={app.busy}>Ouvrir</button>
      </div>
      <p class="hint">
        Base, clés du serveur et pièces jointes : tout est là. Arrête le coffre avant de copier ce dossier pour une
        sauvegarde.
      </p>
    </section>
  </div>

  {#if app.error}<p class="error" role="alert">{app.error}</p>{/if}
</div>
