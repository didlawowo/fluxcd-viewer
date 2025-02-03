#!/bin/bash

# 🚀 Script de déploiement FluxCD avec application de test
# Prérequis: kubectl, flux CLI, git

# 💫 Variables à configurer
GITHUB_USER="didlawowo"
GITHUB_REPO="test-fluxcd-viewer"
GITHUB_BRANCH="main"

# 📦 Installation de FluxCD
echo "🔧 Installation de FluxCD..."
flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=$GITHUB_REPO \
  --branch=$GITHUB_BRANCH \
  --path=clusters/my-cluster \
  --personal \
  --token-auth \
  --private=false
