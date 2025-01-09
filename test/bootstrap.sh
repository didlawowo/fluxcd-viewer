#!/bin/bash

# 🚀 Script de déploiement FluxCD avec application de test
# Prérequis: kubectl, flux CLI, git

# 💫 Variables à configurer
GITHUB_USER="didlawowo"
GITHUB_REPO="test-fluxcd-viwer"
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

# 📂 Création de la structure de base
mkdir -p ./base
mkdir -p ./overlays/dev

# 🎯 Création du déploiement de base (nginx comme exemple)
cat >./base/deployment.yaml <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
        ports:
        - containerPort: 80
EOF

# 🔧 Création du service
cat >./base/service.yaml <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: test-app
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: test-app
EOF

# 📚 Création du kustomization.yaml de base
cat >./base/kustomization.yaml <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml
  - service.yaml
EOF

# 🛠️ Création du kustomization.yaml pour dev
cat >./overlays/dev/kustomization.yaml <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: test-app
resources:
  - ../../base
  - namespace.yaml
patches:
  - patch: |-
      - op: replace
        path: /spec/replicas
        value: 2
    target:
      kind: Deployment
      name: test-app
EOF

# 🌈 Création du namespace
cat >./overlays/dev/namespace.yaml <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: test-app
EOF

# 📡 Création de la source GitRepository
cat >./clusters/my-cluster/test-app-source.yaml <<'EOF'
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: test-app
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/${GITHUB_USER}/${GITHUB_REPO}
  ref:
    branch: ${GITHUB_BRANCH}
EOF

# ⚙️ Création du Kustomization FluxCD
cat >./clusters/my-cluster/test-app-kustomization.yaml <<'EOF'
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: test-app
  namespace: flux-system
spec:
  interval: 5m
  path: ./overlays/dev
  prune: true
  sourceRef:
    kind: GitRepository
    name: test-app
  targetNamespace: test-app
EOF

# 🚀 Commit et push des fichiers
git add .
git commit -m "✨ Initial FluxCD setup with test application"
git push

# 🔍 Vérification du déploiement
echo "⏳ Attente du déploiement..."
sleep 30
flux get kustomizations
kubectl get pods -n test-app
