"""
Statistical profiling for long-running Python processes. This was built to work
with gevent, but would probably work if you ran the emitter in a separate OS
thread too.
Example usage
-------------
Add
>>> gevent.spawn(run_profiler, '0.0.0.0', 16384)
in your program to start the profiler, and run the emitter in a new greenlet.
Then curl localhost:16384 to get a list of stack frames and call counts.
"""

import atexit
import os
import signal
import time

def nanosecond_time():
    return int(time.perf_counter() * 1e9)

class FrameData:
    def __init__(self, frame):
        self._function_name = frame.f_code.co_name
        self._module = frame.f_globals['__name__']

        # Depending on Python version, frame.f_code.co_filename either stores just the file name or the entire absolute path.
        self._file_name = os.path.basename(frame.f_code.co_filename)
        self._abs_path = os.path.abspath(frame.f_code.co_filename) # TODO: Must verify this will give us correct absolute paths in all cases!
        self._line_number = frame.f_lineno

    def __str__(self):
        return f'{self._function_name}({self._module}) in {self._file_name}:{self._line_number}'

class StackSample:
    def __init__(self, top_frame, profiler_start_time):
        self._sample_time = nanosecond_time() - profiler_start_time
        self._stack = []
        self._add_all_frames(top_frame)

    def _add_all_frames(self, top_frame):
        frame = top_frame
        while frame is not None:
            self._stack.append(FrameData(frame))
            frame = frame.f_back

    def __str__(self):
        return f'Time: {self._sample_time}; Stack: {[str(frame) for frame in reversed(self._stack)]}'

class Sampler(object):
    """
    A simple stack sampler for low-overhead CPU profiling: samples the call
    stack every `interval` seconds and keeps track of counts by frame. Because
    this uses signals, it only works on the main thread.
    """
    def __init__(self, interval=0.01):
        self.interval = interval
        self._stack_samples = None

    def start(self):
        self._start_time = nanosecond_time()
        self._stack_samples = []
        try:
            signal.signal(signal.SIGVTALRM, self._sample)
        except ValueError:
            raise ValueError('Can only sample on the main thread')

        signal.setitimer(signal.ITIMER_VIRTUAL, self.interval)
        atexit.register(self.stop)

    def _sample(self, signum, frame):
        self._stack_samples.append(StackSample(frame, self._start_time))
        signal.setitimer(signal.ITIMER_VIRTUAL, self.interval)

    def _format_frame(self, frame):
        return '{}({})'.format(frame.f_code.co_name,
                               frame.f_globals.get('__name__'))

    def __str__(self):
        return '\n'.join([str(sample) for sample in self._stack_samples])

    def stop(self):
        signal.setitimer(signal.ITIMER_VIRTUAL, 0)

    def __del__(self):
        self.stop()
