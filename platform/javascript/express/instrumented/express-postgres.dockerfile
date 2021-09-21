FROM node:14.17.3-slim

WORKDIR /app

COPY package.json ./
RUN npm install
# Some dependencies are not in package.json to be able to reuse cached layers
# from the baseline image.
# See https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#add-or-copy.
RUN npm install @sentry/node@6.11.0
COPY . ./

ENV NODE_ENV production

EXPOSE 8080

CMD ["node", "postgresql-app.js"]
