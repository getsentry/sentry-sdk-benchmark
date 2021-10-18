Sentry.init do |config|
  config.breadcrumbs_logger = [:active_support_logger]
  config.traces_sample_rate = 1.0
  config.send_default_pii = true
end
