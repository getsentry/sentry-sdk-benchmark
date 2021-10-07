FROM node:14.17.3-slim

WORKDIR /app

COPY package.json ./
RUN npm install
COPY . ./

ENV NODE_ENV production

EXPOSE 8080

CMD ["node", "-r", "./tracing.js", "postgresql-app.js"]
