# Fiche projet : Aria Vault

Ce que cette appli a de propre. À relire avant toute modification, et à tenir à
jour quand l'appli change. Fiche écrite après coup, le 18 novembre 2026, d'après
le README, `docs/ARCHITECTURE.md` et le code : ce qui n'a pas pu y être lu reste
« à renseigner ».

| | |
| --- | --- |
| Nom | Aria Vault (dépôt et exe : `Aria-Vault`) |
| Promesse | Ton coffre de mots de passe Vaultwarden, en HTTPS, dans un seul exe Windows. |
| Élément orange du logo | une mallette blindée |
| Fond animé | à renseigner : motif, ce qui est aléatoire, nombre par défaut |
| Stack | Go 1.26, Wails v3 (bêta) ; Vaultwarden et son interface web embarqués dans l'exe |
| Droits administrateur | non ; seule l'autorisation du pare-feu Windows (réseaux privés) peut demander une confirmation |
| Démarrage automatique | oui, proposé : lancement silencieux à l'ouverture de session, avec icône de notification |
| Installation par l'exe | non : exe portable, à décompresser dans un dossier stable |
| Mises à jour intégrées | oui : vérification sur les releases du dépôt, installation en un clic, empreintes SHA-256 vérifiées, relève de l'ancienne version |
| Port local (interface web) | aucune interface web locale : fenêtre Wails. Vaultwarden écoute sur `127.0.0.1:8000` |
| Appli serveur du réseau local | **oui, exception au contrat des applis** : relais HTTPS d'Aria sur le port 443, sur toutes les interfaces, pour les applis Bitwarden du téléphone, du navigateur et du PC. Certificat Let's Encrypt par validation DNS, domaine pointé vers une adresse privée, aucun port ouvert sur la box : jamais exposé sur Internet, joint par le VPN hors de chez toi |
| Données de l'utilisateur | `%USERPROFILE%\.Aria-Vault` (`data\` le coffre, `certificats\`, `config.json`, `aria.log`, `fenetre.json`) ; secrètes : identifiants DNS, clé push, mot de passe e-mail et phrase des sauvegardes, chiffrés par DPAPI dans `config.json` |
| Écritures hors du dossier de données | `%LOCALAPPDATA%\AriaVault` : Vaultwarden et l'interface web réextraits et vérifiés à chaque démarrage, cache de la fenêtre, fichier de relève (**écart au contrat**, qui ne permet rien dans `%LOCALAPPDATA%`) ; l'entrée de démarrage automatique ; l'enregistrement DNS chez le fournisseur choisi ; l'autorisation du pare-feu |
| Connexions sortantes | le fournisseur DNS choisi (DuckDNS, Cloudflare, OVHcloud ou deSEC), Let's Encrypt, des résolveurs DNS publics, GitHub (vérification et téléchargement des mises à jour) ; si tu les actives : le relais push de Bitwarden et ta messagerie. Aucune télémétrie |
| Composants tiers embarqués | Vaultwarden (AGPL-3.0, sources exactes jointes à chaque release), l'interface web Bitwarden (bw_web_builds), les bibliothèques Go compilées dans l'exe : voir `LICENCES-TIERS.txt` |
| Réussite de la v0.1 | à renseigner |

Les lignes « à renseigner » se complètent avant la première ligne de code.
