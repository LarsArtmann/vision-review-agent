import json, urllib.request, urllib.error, ssl, random, re
ctx = ssl.create_default_context()
SITES = [
 "https://art-dupl.lars.software","https://cleanwizard.lars.software","https://cmdguard.web.app",
 "https://dynamicmarkdown.lars.software","https://emeet-pixyd.lars.software","https://atomicwrite.lars.software",
 "https://branded-id.lars.software","https://errorfamily.lars.software","https://filewatcher.lars.software",
 "https://go-output.lars.software","https://go-workflow-auditlog.lars.software","https://gogenfilter.lars.software",
 "https://md-go-validator.lars.software","https://do-auditlog.lars.software","https://typespec-asyncapi.web.app",
 "https://templcomponents.lars.software","https://lars-learnings.web.app",
]
def get(url):
    req = urllib.request.Request(url, headers={"User-Agent":"audit/1.0"})
    try:
        with urllib.request.urlopen(req, timeout=15, context=ctx) as r:
            return r.status, r.read()
    except urllib.error.HTTPError as e:
        return e.code, b""
    except Exception as e:
        return None, str(e).encode()

res = {}
for s in SITES:
    rand = f"nonexistent-{random.randrange(10**9)}"
    code, body = get(f"{s}/{rand}")
    html = body.decode("utf-8","replace")[:2000]
    soft404 = code == 200
    default404 = "Page Not Found" in html and "firebase" in html.lower() or "404 - Page not found" in html
    st, home = get(s)
    h = home.decode("utf-8","replace")
    fav = 'rel="icon"' in h or "rel='icon'" in h or 'rel="shortcut icon"' in h
    manifest = "manifest" in h
    theme = re.search(r'<meta\s+name="theme-color"', h) is not None
    res[s] = {"404code": code, "soft404": soft404, "default404": default404,
              "favicon": fav, "manifest": manifest, "themeColor": theme}
    print(s, res[s], flush=True)
json.dump(res, open("/tmp/vra/audit404_meta.json","w"), indent=1)
