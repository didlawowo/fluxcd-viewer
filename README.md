# 🚀 FluxCD Viewer

FluxCD Viewer est une interface web légère permettant de visualiser et monitorer l'état de vos Kustomizations FluxCD dans votre cluster Kubernetes.

## 🎯 Fonctionnalités

- Vue d'ensemble des Kustomizations FluxCD
- Statut en temps réel des déploiements
- Regroupement par catégories (apis, apps, addons)
- Détails des conditions et messages d'erreur
- Healthcheck endpoint
- Interface responsive et moderne

## 🛠️ Prérequis

- Un cluster Kubernetes avec FluxCD installé
- Un accès kubectl configuré (fichier kubeconfig)

## 📦 Installation

### Option 1 : Docker

\```bash

# Lancer le container avec votre kubeconfig monté

docker run -p 8080:8080 \
 -v ~/.kube/config:/root/.kube/config \
 didlawowo/fluxcd-viewer:latest
\```

### Option 2 : Helm

\```bash

# Ajouter le repo Helm

helm repo add fluxcd-viewer <https://didlawowo.github.io/fluxcd-viewer>
helm repo update

# Installer le chart

helm install fluxcd-viewer fluxcd-viewer/fluxcd-viewer
\```

## 📝 Configuration

L'application utilise les variables d'environnement suivantes :

| Variable     | Description                       | Default          |
| ------------ | --------------------------------- | ---------------- |
| `PORT`       | Port d'écoute du serveur          | `8080`           |
| `KUBECONFIG` | Chemin vers le fichier kubeconfig | `~/.kube/config` |

## 🔍 Utilisation

1. Accédez à l'interface web : `http://localhost:8080`
2. L'interface affiche automatiquement vos Kustomizations
3. Cliquez sur une Kustomization pour voir ses détails

## 🏗️ Développement local

\```bash

# Cloner le repo

git clone <https://github.com/didlawowo/fluxcd-viewer.git>
cd fluxcd-viewer

# Installer les dépendances

go mod download

# Lancer en local

go run main.go
\```

## 🔐 Sécurité

L'application nécessite un accès en lecture seule aux ressources FluxCD. Il est recommandé de créer un ServiceAccount dédié avec les permissions minimales requises.

## 🤝 Contribution

Les contributions sont bienvenues ! N'hésitez pas à ouvrir une issue ou une pull request.

## 📄 Licence

MIT
