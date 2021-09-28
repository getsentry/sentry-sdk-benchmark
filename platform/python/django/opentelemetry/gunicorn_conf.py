import multiprocessing
import os
import sys

_is_pypy = hasattr(sys, 'pypy_version_info')
_is_travis = os.environ.get('TRAVIS') == 'true'

workers = multiprocessing.cpu_count() * 3
if _is_travis:
    workers = 2

bind = "0.0.0.0:8080"
keepalive = 120
errorlog = '-'
pidfile = 'gunicorn.pid'
pythonpath = 'hello'

if _is_pypy:
    worker_class = "tornado"
else:
    worker_class = "meinheld.gmeinheld.MeinheldWorker"

    def post_fork(server, worker):
        # Disalbe access log
        import meinheld.server
        meinheld.server.set_access_logger(None)

        from opentelemetry import trace
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor
        from opentelemetry.exporter.zipkin.json import ZipkinExporter
        from opentelemetry.instrumentation.django import DjangoInstrumentor
        from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor

        # The BatchSpanProcessor is not fork-safe and doesn't work well with
        # application servers (Gunicorn, uWSGI) which are based on the pre-fork
        # web server model.
        # Using the post_fork hook is the suggested workaround.
        # See https://opentelemetry-python.readthedocs.io/en/latest/examples/fork-process-model/README.html
        trace.set_tracer_provider(TracerProvider())
        trace.get_tracer_provider().add_span_processor(
            BatchSpanProcessor(ZipkinExporter())
        )
        # from opentelemetry.sdk.trace.export import SimpleSpanProcessor, ConsoleSpanExporter
        # trace.get_tracer_provider().add_span_processor(
        #     SimpleSpanProcessor(ConsoleSpanExporter())
        # )

        os.environ.setdefault("DJANGO_SETTINGS_MODULE", "hello.settings")
        DjangoInstrumentor().instrument()
        Psycopg2Instrumentor().instrument()
