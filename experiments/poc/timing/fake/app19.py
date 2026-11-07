"""A FAKE app for row 19: a tiny HTTP server whose delay decays with the number of requests served."""
import argparse
import math
import random
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ap = argparse.ArgumentParser()
ap.add_argument("--port", type=int, required=True)
ap.add_argument("--base-ms", type=float, default=1.0)
ap.add_argument("--amp-ms", type=float, default=0.0)
ap.add_argument("--tau", type=float, default=60.0)
ap.add_argument("--jitter-ms", type=float, default=0.0)
a = ap.parse_args()
rng = random.Random(1)
count = [0]


class H(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):
        n = count[0]
        count[0] += 1
        time.sleep((a.base_ms + a.amp_ms * math.exp(-n / a.tau) + rng.random() * a.jitter_ms) / 1000)
        body = b"ok"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass


ThreadingHTTPServer(("127.0.0.1", a.port), H).serve_forever()
