#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Enable HTTPS for a BaoTa-panel Nginx vhost using certbot (webroot).

Usage:
  ./ops/scripts/tencent-enable-https-certbot.sh <domain>

Env:
  PROXY_PORT       default: 3000
  WEBROOT          default: /www/wwwroot/<domain>
  VHOST_CONF       default: /www/server/panel/vhost/nginx/<domain>.conf
  CERTBOT_EMAIL    optional: email used for Let's Encrypt account (recommended)

Notes:
  - This will modify the Nginx vhost conf for the domain.
  - Requires domain DNS to point to this server and NOT be redirected by DNSPod webblock.
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

DOMAIN="${1:-${HOSTNAME:-}}"
if [ -z "$DOMAIN" ]; then
  echo "ERROR: missing domain" >&2
  usage >&2
  exit 2
fi

PROXY_PORT="${PROXY_PORT:-3000}"
WEBROOT="${WEBROOT:-/www/wwwroot/${DOMAIN}}"
VHOST_CONF="${VHOST_CONF:-/www/server/panel/vhost/nginx/${DOMAIN}.conf}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"

mkdir -p "$WEBROOT/.well-known/acme-challenge"

if [ ! -f "$VHOST_CONF" ]; then
  cat >"$VHOST_CONF" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location ^~ /.well-known/acme-challenge/ {
        root ${WEBROOT};
        default_type "text/plain";
    }

    location / {
        proxy_pass http://127.0.0.1:${PROXY_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600;
    }
}
EOF

  nginx -t
  nginx -s reload
fi

TOKEN="codex-acme-preflight-$(date +%s)"
echo -n "$TOKEN" >"$WEBROOT/.well-known/acme-challenge/$TOKEN"

echo "== Preflight (local) ==" >&2
if ! curl -fsS "http://127.0.0.1/.well-known/acme-challenge/${TOKEN}" -H "Host: ${DOMAIN}" | grep -qx "$TOKEN"; then
  rm -f "$WEBROOT/.well-known/acme-challenge/$TOKEN" || true
  echo "ERROR: local Nginx cannot serve ACME challenge path for ${DOMAIN}" >&2
  exit 2
fi

echo "== Preflight (public) ==" >&2
PUBLIC_BODY="$(curl -m 10 -sS "http://${DOMAIN}/.well-known/acme-challenge/${TOKEN}" || true)"
rm -f "$WEBROOT/.well-known/acme-challenge/$TOKEN" || true

if [ "$PUBLIC_BODY" != "$TOKEN" ]; then
  echo "ERROR: public HTTP check failed. Domain may be pointing elsewhere or redirected (e.g. DNSPod webblock)." >&2
  echo "Hint: run 'curl -v http://${DOMAIN}/' and ensure it does NOT redirect to dnspod.qcloud.com webblock." >&2
  exit 2
fi

echo "== Issue certificate via certbot (webroot) ==" >&2
certbot_args=(certbot certonly --webroot -w "$WEBROOT" -d "$DOMAIN" --agree-tos --non-interactive)
if [ -n "$CERTBOT_EMAIL" ]; then
  certbot_args+=(--email "$CERTBOT_EMAIL")
else
  certbot_args+=(--register-unsafely-without-email)
fi
"${certbot_args[@]}"

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
if [ ! -f "$CERT_DIR/fullchain.pem" ] || [ ! -f "$CERT_DIR/privkey.pem" ]; then
  echo "ERROR: certbot finished, but cert files not found under $CERT_DIR" >&2
  exit 2
fi

echo "== Write Nginx HTTPS vhost ==" >&2
cat >"$VHOST_CONF" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location ^~ /.well-known/acme-challenge/ {
        root ${WEBROOT};
        default_type "text/plain";
    }

    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate ${CERT_DIR}/fullchain.pem;
    ssl_certificate_key ${CERT_DIR}/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://127.0.0.1:${PROXY_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600;
    }
}
EOF

mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat >/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
nginx -t
nginx -s reload
EOF
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

nginx -t
nginx -s reload

echo "HTTPS_OK domain=${DOMAIN}"
echo "Test: curl -I https://${DOMAIN}/api/status"
