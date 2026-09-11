# Run only against a fresh disposable docker-compose.test.yml environment.
import json, urllib.request, urllib.error, subprocess, time, base64, hmac, hashlib, struct
from pathlib import Path
import os
compose=['docker','compose','-p','tpm-fix-regression','-f',os.environ.get('TPM_TEST_COMPOSE',str(Path(__file__).resolve().parents[1]/'docker-compose.test.yml'))]
def run(*args): return subprocess.check_output(compose+list(args),text=True).strip()
base='http://'+run('port','backend','4000')+'/api/v1'
proxy='http://'+run('port','tengine','80')
front='http://'+run('port','frontend','80')
def api(method,path,data=None,token=None,want=200):
 headers={'Content-Type':'application/json'}
 if token: headers['Authorization']='Bearer '+token
 req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers,method=method)
 try:
  with urllib.request.urlopen(req,timeout=25) as r: status=r.status; body=r.read()
 except urllib.error.HTTPError as e: status=e.code; body=e.read()
 assert status==want,(method,path,status,body.decode()[:400])
 return json.loads(body) if body else None
def check(name): print('PASS',name,flush=True)
def expect_http(url,status,body=None,host=None):
 # Reload signals are asynchronous; wait for the new worker generation.
 deadline=time.monotonic()+5
 while time.monotonic()<deadline:
  try:
   req=urllib.request.Request(url,headers={} if host is None else {'Host':host})
   try:
    with urllib.request.urlopen(req,timeout=2) as response:
     actual=response.status; content=response.read().decode()
   except urllib.error.HTTPError as err:
    actual=err.code; content=err.read().decode()
   if actual==status and (body is None or content==body): return
  except OSError: pass
  time.sleep(.1)
 raise AssertionError('Tengine response did not converge to expected status/body')

key=run('exec','-T','backend','cat','/app/data/setup.key')
api('POST','/setup',{'email':'admin@example.com','password':'RegressionPassword123!','setup_key':key},want=201)
tokens=api('POST','/auth/login',{'email':'admin@example.com','password':'RegressionPassword123!'})
admin=tokens['access_token']
check('production startup and initial setup')
with urllib.request.urlopen(front+'/login') as r: assert b'<div id="root">' in r.read()
with urllib.request.urlopen(front+'/api/v1/setup/status') as r: assert json.load(r)['setup_required'] is False
check('frontend SPA and API proxy')
user=api('POST','/users',{'email':'user@example.com','password':'RegressionPassword123!','role':'user'},admin,want=201)
ut=api('POST','/auth/login',{'email':'user@example.com','password':'RegressionPassword123!'})['access_token']
api('PUT','/users/'+str(user['id']),{'role':'admin'},ut,want=403)
api('GET','/settings/cloudflare',token=ut,want=403)
check('normal user cannot manage users or settings')
acl=api('POST','/access-lists',{'name':'deny-test','rules':json.dumps([{'action':'allow','ip':'192.0.2.0/24'}])},admin,want=201)
host=api('POST','/proxy-hosts',{'domain':'app.example.com','forward_host':'frontend','forward_port':80,'access_list_id':acl['id']},admin,want=201)
hid=str(host['id'])
def config(domain): return run('exec','-T','tengine','cat','/etc/tengine/conf.d/'+domain+'.conf')
assert 'allow 192.0.2.0/24;' in config('app.example.com')
expect_http(proxy,403,host='app.example.com')
check('ACL applied immediately to a new proxy and enforced by Tengine')
api('POST','/proxy-hosts/'+hid+'/disable',{},admin)
api('PUT','/proxy-hosts/'+hid,{'forward_port':80},admin)
run('exec','-T','tengine','test','!','-e','/etc/tengine/conf.d/app.example.com.conf')
api('POST','/proxy-hosts/'+hid+'/enable',{},admin)
assert 'allow 192.0.2.0/24;' in config('app.example.com')
api('PUT','/proxy-hosts/'+hid,{'domain':'renamed.example.com'},admin)
run('exec','-T','tengine','test','!','-e','/etc/tengine/conf.d/app.example.com.conf')
assert 'allow 192.0.2.0/24;' in config('renamed.example.com')
check('disable/edit/enable and domain rename preserve intended state')
api('PUT','/proxy-hosts/'+hid,{'forward_host':'does-not-exist.invalid'},admin,want=500)
hosts=api('GET','/proxy-hosts',token=admin)
assert hosts['data'][0]['forward_host']=='frontend'
assert 'server frontend:80;' in config('renamed.example.com')
check('real Tengine validation rejection rolls back files and PostgreSQL')
api('PUT','/proxy-hosts/'+hid,{'load_balancing':'consistent_hash','access_list_id':None},admin)
assert 'hash $request_uri consistent;' in config('renamed.example.com')
check('consistent-hash config and clearing ACL pass Tengine validation')
api('POST','/certificates/custom',{'domain':'../../tmp/traversal','cert_content':'invalid','key_content':'invalid'},admin,want=400)
api('POST','/certificates/custom',{'domain':'app.example.com','cert_content':'invalid','key_content':'invalid'},admin,want=400)
cert=api('POST','/certificates/self-signed',{'domain':'renamed.example.com'},admin,want=201)
custom=api('POST','/certificates/custom',{'domain':'renamed.example.com','cert_content':cert['cert_content'],'key_content':cert['key_content']},admin,want=201)
api('PUT','/proxy-hosts/'+hid,{'ssl_enabled':True,'certificate_id':custom['id']},admin)
api('DELETE','/certificates/'+str(custom['id']),token=admin,want=409)
check('certificate validation, safe upload, SSL config and in-use deletion protection')
body="<h1>It's $5</h1>\n<p>Literal \\ and quotes</p>"
api('PUT','/settings/default-server',{'status':'200','body':body},admin)
assert api('GET','/settings/default-server',token=admin)['body']==body
expect_http(proxy,200,body=body)
api('PUT','/settings/default-server',{'status':'444','body':''},admin)
assert api('GET','/settings/default-server',token=admin)['status']=='444'
check('default HTML response preserves quotes and dollar signs')
time.sleep(7)
twofa=api('POST','/auth/2fa/setup',{},admin)
secret=base64.b32decode(twofa['secret']+'='*((-len(twofa['secret']))%8))
digest=hmac.new(secret,struct.pack('>Q',int(time.time())//30),hashlib.sha1).digest(); offset=digest[-1]&15
code=str((struct.unpack('>I',digest[offset:offset+4])[0]&0x7fffffff)%1000000).zfill(6)
api('POST','/auth/2fa/verify',{'code':code},admin)
temp=api('POST','/auth/login',{'email':'admin@example.com','password':'RegressionPassword123!'})['temp_token']
api('GET','/users',token=temp,want=401)
api('POST','/auth/refresh',{'refresh_token':temp},want=401)
api('GET','/users',token=tokens['refresh_token'],want=401)
check('2FA temporary and refresh tokens cannot access protected API')
# Cached protection remains effective before any network refresh after restart.
run('stop','backend')
run('exec','-T','postgres','psql','-U','tpm','-d','tpm_test','-c',"UPDATE cloudflare_settings SET enabled=true, ipv4_list='192.0.2.0/24', ipv6_list='';")
run('start','backend')
base='http://'+run('port','backend','4000')+'/api/v1'
for _ in range(40):
 try:
  api('GET','/setup/status'); break
 except (OSError,AssertionError): time.sleep(.25)
else: raise AssertionError('Backend did not become ready after restart')
geo=run('exec','-T','backend','cat','/etc/tengine/conf.d/cloudflare-geo.conf')
assert 'default 0;' in geo and '192.0.2.0/24 1;' in geo
assert key==run('exec','-T','backend','cat','/app/data/setup.key')
check('restart restores cached Cloudflare protection and keeps setup key')
print('ALL END-TO-END CHECKS PASSED',flush=True)
