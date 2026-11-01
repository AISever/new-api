# Isolated synthetic fixtures only. Never point these scripts at production.
# Loopback-only, task-owned bridge into the internal fixture network.
import http.client,http.server,json,pathlib
root=pathlib.Path(__file__).resolve().parents[3]/'.env/artifacts/invitation-test-20261101';root.mkdir(parents=True,exist_ok=True);target=json.loads((root/'ui-address.json').read_text())
class Proxy(http.server.BaseHTTPRequestHandler):
 protocol_version='HTTP/1.1'
 def do_GET(self):self.proxy()
 def do_POST(self):self.proxy()
 def do_PUT(self):self.proxy()
 def do_DELETE(self):self.proxy()
 def do_OPTIONS(self):self.proxy()
 def proxy(self):
  connection=http.client.HTTPConnection(target['target'],3000,timeout=60)
  try:
   body=self.rfile.read(int(self.headers.get('Content-Length','0')))
   headers={k:v for k,v in self.headers.items() if k.lower() not in ['connection','transfer-encoding']}
   connection.request(self.command,self.path,body,headers);r=connection.getresponse();content=r.read()
   self.send_response(r.status)
   for k,v in r.getheaders():
    if k.lower() not in ['connection','transfer-encoding','content-length']:self.send_header(k,v)
   self.send_header('Content-Length',str(len(content)));self.end_headers();self.wfile.write(content)
  finally:connection.close()
 def log_message(self,*args):pass
http.server.ThreadingHTTPServer(('127.0.0.1',13019),Proxy).serve_forever()
