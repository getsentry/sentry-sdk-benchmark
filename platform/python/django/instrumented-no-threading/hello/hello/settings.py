import os
import sentry_sdk
from sentry_sdk.integrations.django import DjangoIntegration

from sentry_sdk.integrations.logging import LoggingIntegration
from sentry_sdk.integrations.stdlib import StdlibIntegration
from sentry_sdk.integrations.excepthook import ExcepthookIntegration
from sentry_sdk.integrations.dedupe import DedupeIntegration
from sentry_sdk.integrations.atexit import AtexitIntegration
from sentry_sdk.integrations.modules import ModulesIntegration
from sentry_sdk.integrations.argv import ArgvIntegration

sentry_sdk.init(
    traces_sample_rate=1.0,
    send_default_pii=True,
    default_integrations=False,
    # See: https://github.com/getsentry/sentry-python/blob/54bc81cfb68d4c1df752d2358b8caf1969f1490d/sentry_sdk/integrations/__init__.py#L69
    integrations=[
        DjangoIntegration(),
        LoggingIntegration(),
        StdlibIntegration(),
        ExcepthookIntegration(),
        DedupeIntegration(),
        AtexitIntegration(),
        ModulesIntegration(),
        ArgvIntegration(),
        # Remove threading integration
        # sentry_sdk.integrations.threading.ThreadingIntegration()
    ],
    # debug=True,
)

DEBUG = False

SECRET_KEY = '_7mb6#v4yf@qhc(r(zbyh&amp;z_iby-na*7wz&amp;-v6pohsul-d#y5f'
ADMINS = ()

MANAGERS = ADMINS

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.' + os.environ['DJANGO_DB'], # Add 'postgresql_psycopg2', 'mysql', 'sqlite3' or 'oracle'.
        'NAME': 'hello_world',           # Or path to database file if using sqlite3.
        'USER': 'benchmarkdbuser',       # Not used with sqlite3.
        'PASSWORD': 'benchmarkdbpass',   # Not used with sqlite3.
        'HOST': 'tfb-database',  # Set to empty string for localhost. Not used with sqlite3.
        'PORT': '',                      # Set to empty string for default. Not used with sqlite3.
        'CONN_MAX_AGE': 30,
    }
}

TIME_ZONE = 'America/Chicago'
LANGUAGE_CODE = 'en-us'
USE_I18N = False
USE_L10N = False
USE_TZ = False

MEDIA_ROOT = ''
MEDIA_URL = ''
STATIC_ROOT = ''
STATIC_URL = '/static/'
STATICFILES_DIRS = ()
STATICFILES_FINDERS = ()
MIDDLEWARE = ()

ROOT_URLCONF = 'hello.urls'
WSGI_APPLICATION = 'hello.wsgi.application'

TEMPLATES = [
    {
        'BACKEND': 'django.template.backends.django.DjangoTemplates',
        'DIRS': [],
        'APP_DIRS': True,
        'OPTIONS': {},
    },
]

INSTALLED_APPS = (
    'django.contrib.contenttypes',
    'django.contrib.sessions',
    'world',
)

LOGGING = {
    'version': 1,
    'disable_existing_loggers': True,
    'handlers': {},
    'loggers': {},

}

ALLOWED_HOSTS = ['*']
