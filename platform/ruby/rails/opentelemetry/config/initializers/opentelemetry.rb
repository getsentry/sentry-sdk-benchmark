OpenTelemetry::SDK.configure do |c|
  c.use_all

  c.add_span_processor(
    OpenTelemetry::SDK::Trace::Export::BatchSpanProcessor.new(
      OpenTelemetry::Exporter::Zipkin::Exporter.new
    )
  )
end
