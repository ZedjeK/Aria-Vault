// État partagé de l'interface et appels au backend Go (VaultService).
import { Call } from '@wailsio/runtime'

const call = (method, ...args) => Call.ByName(`main.VaultService.${method}`, ...args)
const message = (err) => String(err?.message ?? err)

export const app = $state({
  page: 'home',
  status: null,
  loadError: '',
  busy: false,
  error: '', // erreur de la dernière action
  journal: [],
})

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
    if (app.page === 'journal' || app.status?.moteur?.etat === 'demarrage') refreshJournal()
  }, 2000)
}

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

export const start = () => run(() => call('Start'))
export const stop = () => run(() => call('Stop'))
export const restart = () => run(() => call('Restart'))
export const setInscriptions = (on) => run(() => call('SetInscriptions', on))
export const setAutostart = (on) => run(() => call('SetAutostart', on))
export const openVault = () => run(() => call('OpenVault'))
export const openData = () => run(() => call('OpenDataFolder'))

export async function saveConfig(input) {
  const res = await run(() => call('SaveConfig', input))
  if (res?.error) app.error = res.error
  return res ?? { error: app.error }
}

// ---------- Mise en forme ----------

// Go envoie une date nulle pour « jamais » (année 1).
const valid = (iso) => iso && new Date(iso).getFullYear() > 2000

export function formatDate(iso) {
  if (!valid(iso)) return ''
  return new Date(iso).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })
}

export function formatSince(iso) {
  if (!valid(iso)) return ''
  const s = Math.max(0, Math.round((Date.now() - new Date(iso)) / 1000))
  if (s < 60) return 'à l\'instant'
  if (s < 3600) return `depuis ${Math.round(s / 60)} min`
  if (s < 86400) return `depuis ${Math.floor(s / 3600)} h ${Math.round((s % 3600) / 60)} min`
  return `depuis ${Math.floor(s / 86400)} j`
}

export function daysLeft(iso) {
  if (!valid(iso)) return null
  return Math.floor((new Date(iso) - Date.now()) / 86400000)
}
