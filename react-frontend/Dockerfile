FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
ARG VITE_API_URL=http://localhost:3000
ARG VITE_AUTH_USERNAME=admin
ARG VITE_AUTH_PASSWORD=ChangeMe123@
ARG VITE_API_KEY=dev-local-api-key
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_AUTH_USERNAME=$VITE_AUTH_USERNAME
ENV VITE_AUTH_PASSWORD=$VITE_AUTH_PASSWORD
ENV VITE_API_KEY=$VITE_API_KEY
RUN npm run build

FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
