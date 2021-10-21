FROM ruby:3.0

# throw errors if Gemfile has been modified since Gemfile.lock
RUN bundle config --global frozen 1

EXPOSE 8080
WORKDIR /rails

COPY ./Gemfile* /rails/

ENV BUNDLE_WITHOUT=mysql
RUN bundle install --jobs=8

COPY . /rails/

# Extra dependencies installed separately to reuse build cache layers from the
# baseline app.
RUN bundle add sentry-rails --source 'https://rubygems.org' --version '4.7.3'

ENV RAILS_ENV=production_postgresql
ENV PORT=8080
CMD ["rails", "server"]
