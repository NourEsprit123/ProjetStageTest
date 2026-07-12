# --- Étape 1 : installation des dépendances et build ---
FROM node:20-alpine AS builder

WORKDIR /app

# ### AJOUTÉ : On déclare que le build a besoin de cette variable
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY package.json package-lock.json* ./
RUN npm install

COPY . .
RUN npm run build

# --- Étape 2 : image finale pour exécuter l'app ---
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
# ### AJOUTÉ : On garde la variable pour le runtime
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json

EXPOSE 3000

CMD ["npm", "start"]