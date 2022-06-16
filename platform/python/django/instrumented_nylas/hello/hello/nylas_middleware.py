import hello.nylas_profiler

class NylasMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response
    
    def __call__(self, request):
        profiler = nylas_profiler.Sampler()
        profiler.start()
        response = self.get_response(request)
        profiler.stop()
        
        return response
