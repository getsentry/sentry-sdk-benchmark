// tracing.js

'use strict'

const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { ZipkinExporter } = require("@opentelemetry/exporter-zipkin");
const { BatchSpanProcessor } = require('@opentelemetry/sdk-trace-base');
const { registerInstrumentations } = require('@opentelemetry/instrumentation');

const provider = new NodeTracerProvider();

const zipkinExporter = new ZipkinExporter({
  url: process.env.OTEL_EXPORTER_ZIPKIN_ENDPOINT,
  serviceName: 'movies-service'
})

provider.addSpanProcessor(new BatchSpanProcessor(zipkinExporter));
provider.register();

registerInstrumentations({
  instrumentations: [
    getNodeAutoInstrumentations(),
  ],
});

