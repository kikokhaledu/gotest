FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --omit=dev

COPY . .

ENV PORT=3000
ENV GO_BACKEND_URL=http://go-backend:8080
ENV NODE_ENV=production

EXPOSE 3000

CMD ["npm", "start"]
