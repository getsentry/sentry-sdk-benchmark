FROM node:14.17.3-slim

WORKDIR /app

COPY package.json ./
RUN npm install
# Some dependencies are not in package.json to be able to reuse cached layers
# from the baseline image.
# See https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#add-or-copy.
RUN npm install @opentelemetry/api@1.0.3 @opentelemetry/auto-instrumentations-node@0.25.0 @opentelemetry/exporter-zipkin@1.0.0 @opentelemetry/sdk-node@0.26.0
COPY . ./

ENV NODE_ENV production

EXPOSE 8080

CMD ["node", "-r", "./tracing.js", "postgresql-app.js"]
