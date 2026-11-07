// État partagé de l'interface et appels au backend Go.
//
// Chaque action de l'interface appelle une méthode exportée du VaultService Go
// (src/vaultservice.go, sauvegardes.go, services.go) par son nom. L'état est un
// objet Svelte réactif ($state) : toute page qui lit app.status se met à jour
// quand il change. Le Go reste la source de vérité : après chaque action,
// l'état est relu, et il l'est aussi toutes les 2 secondes.
import { Call } from '@wailsio/runtime'

const call = (method, ...args) => Call.ByName(`main.VaultService.${method}`, ...args)
const message = (err) => String(err?.message ?? err)

export const app = $state({
  page: 'home', // page affichée : home, https, services, sauvegardes, reglages, journal
  status: null, // état complet renvoyé par VaultService.Status (voir View en Go)
  loadError: '', // le backend ne répond pas
  busy: false, // une action est en cours : boutons désactivés, actualisation suspendue
  error: '', // erreur de la dernière action
  journal: [],
  backups: null, // page Sauvegardes
  fournisseurs: [], // fournisseurs DNS et leurs tutoriels
  installation: null, // null, ou pourcentage téléchargé de la mise à jour
})

// ---------- Lecture de l'état ----------

// Les fournisseurs DNS ne changent pas pendant la vie de l'appli : lus une fois.
export async function loadFournisseurs() {
  if (app.fournisseurs.length) return
  try {
    app.fournisseurs = (await call('Fournisseurs')) ?? []
  } catch (err) {
    app.error = message(err)
  }
}

export async function refresh() {
  try {
    app.status = await call('Status')
    app.loadError = ''
  } catch (err) {
    app.loadError = message(err)
  }
}

export async function refreshJournal() {
  try {
    app.journal = (await call('Journal')) ?? []
  } catch {}
}

// L'état change tout seul (démarrage, certificat, plantage) : on le suit.
let timer = null
export function startPolling() {
  if (timer) return
  timer = setInterval(() => {
    if (app.busy) return
    refresh()
    if (app.page === 'journal' || app.page === 'home' || app.status?.moteur?.etat === 'demarrage') refreshJournal()
  }, 2000)
}

// run encadre une action : interface occupée pendant l'appel, erreur affichée
// s'il échoue, puis état et journal relus.
async function run(fn) {
  app.busy = true
  app.error = ''
  try {
    return await fn()
  } catch (err) {
    app.error = message(err)
  } finally {
    app.busy = false
    await refresh()
    refreshJournal()
  }
}

// saveForm : pour les formulaires, le Go renvoie { fields, error } (erreurs par
// champ) au lieu d'échouer.
async function saveForm(method, input) {
  const res = await run(() => call(method, input))
  if (res?.error) app.error = res.error
  return res ?? { error: app.error }
}

// ---------- Coffre, démarrage et veille (pages Accueil et Réglages) ----------
// Chaque fonction appelle la méthode Go du même nom.

export const start = () => run(() => call('Start'))
export const stop = () => run(() => call('Stop'))
export const setInscriptions = (on) => run(() => call('SetInscriptions', on))
export const setAutostart = (on) => run(() => call('SetAutostart', on))
export const setEmpecherVeille = (on) => run(() => call('SetEmpecherVeille', on))
export const openVault = () => run(() => call('OpenVault'))
export const openData = () => run(() => call('OpenDataFolder'))

// Les liens des tutoriels s'ouvrent dans le navigateur, sans bloquer
// l'interface. Le Go n'ouvre que les liens qu'il connaît.
export async function openLink(url) {
  try {
    await call('OpenLink', url)
  } catch (err) {
    app.error = message(err)
  }
}

// ---------- HTTPS et IP publiée ----------

export const saveConfig = (input) => saveForm('SaveConfig', input)
export const setIPAuto = (on) => run(() => call('SetIPAuto', on))
export const checkIP = () => run(() => call('CheckIP'))

// ---------- Mises à jour ----------

export const setVerifMaj = (on) => run(() => call('SetVerifMaj', on))
export const checkUpdate = () => run(() => call('CheckUpdate'))
export const openRelease = () => run(() => call('OpenRelease'))

// Installe la nouvelle version : téléchargement vérifié, remplacement de l'exe,
// puis la nouvelle version prend le relais et cette fenêtre se ferme. La
// progression arrive par l'événement « maj-progression » (voir App.svelte).
export async function installUpdate() {
  app.installation = 0
  const ok = await run(async () => {
    await call('InstallUpdate')
    return true
  })
  app.installation = ok ? 100 : null
  return ok
}

// ---------- Services : synchronisation instantanée et e-mails ----------

export const savePush = (input) => saveForm('SavePush', input)
export const saveSMTP = (input) => saveForm('SaveSMTP', input)
export const disablePush = () => run(() => call('DisablePush'))
export const disableSMTP = () => run(() => call('DisableSMTP'))
export const setInvitations = (on) => run(() => call('SetInvitations', on))

// Renvoie true si l'e-mail de test est parti (l'erreur, sinon, est dans app.error).
export async function testSMTP(destinataire) {
  return run(async () => {
    await call('TestSMTP', destinataire)
    return true
  })
}

// ---------- Sauvegardes ----------

export async function loadBackups() {
  try {
    app.backups = await call('Backups')
  } catch (err) {
    app.error = message(err)
  }
}

// runBackup : comme run, puis la liste des sauvegardes est relue.
async function runBackup(fn) {
  const res = await run(fn)
  await loadBackups()
  return res
}

export const backupNow = () => runBackup(() => call('Backup'))
export const setBackupEncryption = (on, phrase = '') => runBackup(() => call('SetBackupEncryption', on, phrase))
export const importVault = () => runBackup(() => call('ImportVault'))
export const chooseBackupFolder = () => runBackup(() => call('ChooseBackupFolder'))
export const openBackupFolder = () => run(() => call('OpenBackupFolder'))
export const setDailyBackup = (on) => runBackup(() => call('SetDailyBackup', on))
export const exportSettings = (phrase) => run(() => call('ExportSettings', phrase))

// Accueil et Ctrl+S : sans dossier choisi, la page Sauvegardes le fait choisir.
export async function sauvegarderMaintenant() {
  if (!app.status?.configure) return
  if (!app.backups) await loadBackups()
  if (!app.backups?.dossier) {
    app.page = 'sauvegardes'
    return
  }
  return backupNow()
}

// Restaure une sauvegarde. Pour une sauvegarde chiffrée, le Go peut répondre
// { besoinPhrase: true } : la page demande alors la phrase et rappelle.
export async function restoreBackup(nom, phrase = '') {
  const res = await runBackup(() => call('Restore', nom, phrase))
  if (res?.error) app.error = res.error
  return res ?? { error: app.error }
}

export async function importSettings(phrase) {
  const res = await run(() => call('ImportSettings', phrase))
  if (res?.error) app.error = res.error
  return res
}

// ---------- Mise en forme ----------

// Go envoie une date nulle pour « jamais » (année 1).
const valid = (iso) => iso && new Date(iso).getFullYear() > 2000

// « 7 novembre 2026 »
export function formatDate(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
}

// « 14:05 »
export function formatHeure(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}

// « 7 nov. 2026, 14:05 »
export function formatDateHeure(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleString('fr-FR', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// « depuis 3 h 12 min », « depuis 4 j »
export function formatSince(iso) {
  if (!valid(iso)) return ''
  const s = Math.max(0, Math.round((Date.now() - new Date(iso)) / 1000))
  if (s < 60) return 'à l\'instant'
  if (s < 3600) return `depuis ${Math.round(s / 60)} min`
  if (s < 86400) return `depuis ${Math.floor(s / 3600)} h ${Math.round((s % 3600) / 60)} min`
  return `depuis ${Math.floor(s / 86400)} j`
}

// Jours restants avant une date (validité du certificat), null si inconnue.
export function daysLeft(iso) {
  if (!valid(iso)) return null
  return Math.floor((new Date(iso) - Date.now()) / 86400000)
}

// « 412 Ko », « 3,2 Mo »
export function formatTaille(n) {
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1).replace('.', ',')} Mo`
  if (n >= 1 << 10) return `${Math.round(n / (1 << 10))} Ko`
  return `${n} o`
}
