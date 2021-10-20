FROM ruby:3.0

# throw errors if Gemfile has been modified since Gemfile.lock
RUN bundle config --global frozen 1

EXPOSE 8080
WORKDIR /rails

COPY ./Gemfile* /rails/

ENV BUNDLE_WITHOUT=mysql
RUN bundle install --jobs=8

COPY . /rails/

ENV RAILS_ENV=production_postgresql

RUN bundle add opentelemetry-sdk --source 'https://rubygems.org' --version '1.0.0' && \
    bundle add opentelemetry-instrumentation-rails --source 'https://rubygems.org' --version '0.19.4' && \
    bundle add opentelemetry-exporter-zipkin --source 'https://rubygems.org' --version '0.19.2'

ENV PORT=8080
CMD ["rails", "server"]
