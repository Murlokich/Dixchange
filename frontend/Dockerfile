FROM node:24-alpine AS build

WORKDIR /src

# Dependencies first, so code changes don't invalidate the install layer.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# Vite inlines the VITE_* values from envs/.env into the bundle, the same file
# `npm run dev` uses, so changing them means rebuilding the image. Without the file
# the build would succeed and the app fail in the browser, so fail here instead.
RUN test -f envs/.env || { echo "frontend/envs/.env is missing: copy it from envs/.env.example" >&2; exit 1; }
RUN npm run build

# The build output is static files: nginx serves them, no Node at run time.
FROM nginx:1.28-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html

EXPOSE 80
