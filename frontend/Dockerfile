FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci --no-audit --no-fund
COPY . .
# Leave blank when Nginx proxies /api to the backend (the docker-compose default).
ARG VITE_API_URL=
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

FROM nginx:1.27-alpine
# The official Nginx image renders templates at container startup. Keep this
# configurable so local Compose can use `backend` while Railway can use its
# private service DNS name and assigned port.
ENV BACKEND_UPSTREAM=http://backend:8080 \
    PORT=80
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
