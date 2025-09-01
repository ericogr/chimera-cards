# Multi-stage Dockerfile for the React frontend.
# 1) Build the static assets with Node
# 2) Build a tiny Go static-file server and produce a distroless final image

### Build React static files
FROM node:18-bullseye AS node-builder
WORKDIR /src

# Copy package metadata and install dependencies
# When building inside the frontend directory the package files reside here
COPY package*.json ./
RUN npm ci --silent

# Copy the rest of the frontend source and build
COPY . ./

# Accept common build args for CRA apps (these must be passed with --build-arg)
ARG REACT_APP_GOOGLE_CLIENT_ID
ARG REACT_APP_API_BASE_URL
ENV REACT_APP_GOOGLE_CLIENT_ID=$REACT_APP_GOOGLE_CLIENT_ID
ENV REACT_APP_API_BASE_URL=$REACT_APP_API_BASE_URL

ENV CI=true
RUN npm run build


### (no extra server stage) static assets will be served by nginx in the final image
### Final image: use nginx to serve static assets (safer for production web serving)
FROM nginx:stable-alpine

# Copy built static assets from node build stage into nginx's html folder
COPY --from=node-builder /src/build /usr/share/nginx/html

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Ensure permissions (nginx in this image runs as 'nginx' user for worker processes)
RUN chown -R nginx:nginx /usr/share/nginx/html || true

EXPOSE 80

CMD ["/usr/sbin/nginx", "-g", "daemon off;"]
