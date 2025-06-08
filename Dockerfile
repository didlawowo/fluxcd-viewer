FROM python:3.12-alpine3.20

# Créer un utilisateur non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# # Installation des dépendances système
# RUN apk update && apk upgrade && \
#     apk add --no-cache gcc musl-dev # 🔧 Nécessaire pour certaines compilations Python

WORKDIR /app

# Installation des dépendances Python
COPY requirements.txt .
RUN pip install --upgrade pip uv && \
    uv pip install --system -r requirements.txt && \
    rm -rf /root/.cache/pip /root/.cache/uv # 🧹 Nettoyage du cache

# Copie des fichiers application
COPY templates /app/templates
COPY static /app/static
COPY app.py urls.txt ./

# Configuration des permissions
RUN chown -R appuser:appgroup /app && \
    chmod -R 755 /app

# Passage à l'utilisateur non-root
USER appuser

EXPOSE 5000

CMD ["hypercorn", "app:app", "--bind", "0.0.0.0:5000", "--worker-class", "asyncio"]
