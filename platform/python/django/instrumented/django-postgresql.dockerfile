FROM python:3.9.1-buster

WORKDIR /django

COPY requirements.txt ./
RUN pip install -r requirements.txt
COPY requirements-sentry.txt ./
RUN pip install -r requirements-sentry.txt
COPY . ./

EXPOSE 8080

CMD gunicorn --pid=gunicorn.pid hello.wsgi:application -c gunicorn_conf.py --env DJANGO_DB=postgresql
