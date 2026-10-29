**Aria Vault** {VERSION} : ton coffre Vaultwarden en HTTPS dans un seul exe Windows, compatible avec les applis Bitwarden (téléphone, navigateur, PC).

## Installation

1. Télécharge `Aria-Vault-{VERSION}.zip` ci-dessous et décompresse-le dans un dossier qui ne bougera pas (pas Téléchargements).
2. Lance `Aria-Vault.exe`. L'exe n'est pas encore signé ([Code signing policy]({REPO_URL}/blob/main/docs/SIGNATURE.md)) : si Windows SmartScreen affiche « Windows a protégé votre ordinateur », clique sur « Informations complémentaires » puis « Exécuter quand même ».
3. Page **HTTPS** : ton fournisseur DNS (DuckDNS, Cloudflare, OVHcloud ou deSEC), ton domaine et tes identifiants, avec le tuto pas à pas à côté. Puis réserve l'adresse IP du PC sur ta box (tuto Freebox, Livebox, Bbox, SFR dans la même page). Le guide complet est dans le [README]({REPO_URL}#readme).

**Mise à jour depuis une version précédente** : lance simplement la nouvelle version, l'ancienne s'arrête proprement et lui laisse la place. Ton coffre, tes réglages et tes sauvegardes sont conservés.

## Contenu

| Composant | Version | Source |
|---|---|---|
| Vaultwarden | {VW_VERSION} | compilé depuis le source officiel, sans modification : [dani-garcia/vaultwarden@{VW_COMMIT}](https://github.com/dani-garcia/vaultwarden/tree/{VW_COMMIT}) (AGPL-3.0) |
| Interface web | {WEBVAULT} | [dani-garcia/bw_web_builds](https://github.com/dani-garcia/bw_web_builds/releases/tag/{WEBVAULT}), vérifiée par SHA-256 (GPL-3.0) |

Le détail (commit d'Aria, toolchains) est dans `BUILDINFO.txt`, à l'intérieur du zip.

## Vérifier ce que tu as téléchargé

Attestation de provenance Sigstore (nécessite le [CLI GitHub](https://cli.github.com)) :

```powershell
gh attestation verify .\Aria-Vault-{VERSION}.zip --owner {OWNER}
gh attestation verify .\Aria-Vault.exe --owner {OWNER}
```
