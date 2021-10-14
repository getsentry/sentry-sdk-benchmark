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
ENV PORT=8080
CMD ["rails", "server"]
