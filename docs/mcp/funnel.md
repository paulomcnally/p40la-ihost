---
title: "Tailscale Funnel"
sidebar_position: 10
description: "Runbook para publicar el MCP remoto en HTTPS 443 con Tailscale Funnel"
---

# Tailscale Funnel

Funnel publica un nodo de la tailnet en **HTTPS 443** con un certificado gestionado por Tailscale y proxya el tráfico a un servicio local. Es la pieza que permite que claude.ai alcance `https://<nodo>.<tailnet>.ts.net/mcp` sin abrir puertos en el router ni exponer la IP del iHost.

> **Este repositorio es público.** Nunca pegues el nombre real de tu tailnet, el FQDN real `*.ts.net`, direcciones CGNAT (`100.64/10`) o privadas (`192.168/16`), tokens, hashes, API keys ni capturas con esos datos. En esta guía se usan `https://demo-node.example.ts.net` y `192.0.2.10` como placeholders sintéticos.

## Qué hace y qué no hace

| Hace | No hace |
|---|---|
| Termina TLS en 443 con certificado válido | Autenticar usuarios: eso lo hace OAuth 2.1 |
| Proxya a `http://127.0.0.1:8090` | Aislar el endpoint de Internet: queda público |
| Publica un hostname estable de la tailnet | Reemplazar el rate limit, la auditoría ni la revocación |

Funnel **no** debe considerarse una capa de seguridad. La seguridad del endpoint depende de [OAuth 2.1](oauth.md), del doble gate de escritura y de la auditoría.

## Prerrequisitos

1. **Tailscale instalado y el nodo dentro de la tailnet.** El iHost ya lo tiene; en una laptop, instalalo y autenticá el nodo.
2. **HTTPS habilitado en la tailnet.** En la consola de administración, activá *HTTPS Certificates* y MagicDNS. Funnel necesita emitir un certificado para `<nodo>.<tailnet>.ts.net`.
3. **Atributo `funnel` en la política de la tailnet.** En *Access controls*, agregá el nodo o su tag a `nodeAttrs`:

   ```json
   {
     "nodeAttrs": [
       {
         "target": ["tag:p40la-mcp"],
         "attr": ["funnel"]
       }
     ]
   }
   ```

   Reemplazá `tag:p40la-mcp` por el tag o nodo real. Sin este atributo, `tailscale funnel` falla con un error de permisos.
4. **El MCP en modo HTTP escuchando en loopback**, por ejemplo `127.0.0.1:8090`, con `P40LA_MCP_PUBLIC_URL` igual al FQDN que vas a publicar. Ver [http.md](http.md).
5. **El puerto 443 libre en ese nodo.** Funnel solo puede escuchar en 443, 8443 o 10000; los conectores de Claude esperan 443.

### Conflicto con el dashboard en 443

El add-on de p40la-ihost suele publicar el dashboard con `tailscale serve` en 443 (`deploy/tailscale/serve.json`, `AllowFunnel: false`, proxy a `127.0.0.1:8088`).

`tailscale funnel --bg 8090` intenta tomar el handler `/` de 443 hacia el MCP; según la versión de Tailscale, lo reemplaza o falla por conflicto con el handler existente. Si el mismo nodo ya sirve el dashboard en 443:

- Preferí un **nodo dedicado** para el MCP (una laptop o un segundo hostname), o
- configurá handlers por path para `/mcp` y todos los endpoints OAuth (`/.well-known/*`, `/register`, `/authorize`, `/token`), lo que es más frágil y requiere mantener ambas configuraciones.

No publiques el listener administrativo (`8091`) ni el servidor web (`8088`) por Funnel. El dashboard no necesita estar en Internet para que Claude use el MCP.

## Publicar

### Opción A: comando directo

```bash
# Publica 443 → localhost:8090 y queda activo en segundo plano
tailscale funnel --bg 8090

# Estado de Funnel y de Serve
tailscale funnel status
tailscale serve status
```

`--bg` deja la configuración activa después de cerrar la sesión SSH. El comando imprime la URL pública; verificá que coincida exactamente con `P40LA_MCP_PUBLIC_URL`.

### Opción B: configuración declarativa

Si el add-on de Tailscale carga `serve.json` desde el repo, usá la variante con `AllowFunnel: true`:

```json
{
  "TCP": {
    "443": {
      "HTTPS": true
    }
  },
  "Web": {
    "demo-node.example.ts.net:443": {
      "Handlers": {
        "/": {
          "Proxy": "http://127.0.0.1:8090"
        }
      }
    }
  },
  "AllowFunnel": {
    "demo-node.example.ts.net:443": true
  }
}
```

El archivo del repo (`deploy/tailscale/serve-funnel.example.json`) usa `${TS_CERT_DOMAIN}` como placeholder del FQDN. No apliques el comando y el JSON a la vez: son dos formas de configurar lo mismo.

## Verificar

Hacelo **desde fuera de la tailnet** (por ejemplo, datos móviles). Desde dentro de la tailnet, `tailscale serve` puede responder aunque Funnel esté caído y enmascarar el problema.

```bash
# 1. Discovery público
curl -sS https://demo-node.example.ts.net/.well-known/oauth-protected-resource
# {"resource":"https://demo-node.example.ts.net/mcp","authorization_servers":["https://demo-node.example.ts.net"],...}

# 2. /mcp sin token debe dar 401 con el challenge
curl -sS -o /dev/null -w '%{http_code}\n' https://demo-node.example.ts.net/mcp
# 401

curl -sSI https://demo-node.example.ts.net/mcp | grep -i '^www-authenticate'
# WWW-Authenticate: Bearer resource_metadata="https://demo-node.example.ts.net/.well-known/oauth-protected-resource"

# 3. El listener administrativo NO debe ser alcanzable desde Internet
curl -sS -o /dev/null -w '%{http_code}\n' https://demo-node.example.ts.net/healthz
# 404
```

Checklist de verificación:

- [ ] `tailscale funnel status` muestra el handler en 443 hacia `127.0.0.1:8090`.
- [ ] El certificado es válido y el redirect HTTP → HTTPS funciona.
- [ ] `P40LA_MCP_PUBLIC_URL` coincide con el FQDN publicado, sin `/mcp`.
- [ ] Discovery responde `200` y `resource` apunta a `/mcp`.
- [ ] `/mcp` sin token responde `401` con `WWW-Authenticate`.
- [ ] `/healthz` y `/admin/revoke` responden `404` desde Internet.
- [ ] El conector de claude.ai completa OAuth y ejecuta una tool de lectura.

## Rollback

```bash
# 1. Desactivar Funnel
tailscale funnel reset

# 2. Confirmar que ya no hay handlers públicos
tailscale funnel status

# 3. Restaurar el dashboard solo dentro de la tailnet, si ese nodo lo servía
tailscale serve --bg 8088
tailscale serve status
```

`tailscale funnel reset` borra la configuración de Funnel. Si el nodo también servía el dashboard en 443, revisá `tailscale serve status` y volvé a aplicar `deploy/tailscale/serve.json` o el comando `tailscale serve` correspondiente.

Después del rollback:

```bash
# Revocar tokens para que un conector viejo no siga vigente si el endpoint vuelve
curl -s -X POST http://127.0.0.1:8091/admin/revoke

# Opcional: detener el contenedor del MCP
docker stop p40la-mcp
```

Si el objetivo es retirar el conector, eliminá también la integración en claude.ai. Los tokens revocados no se pueden reutilizar aunque Funnel vuelva a publicarse.

## Advertencia de exposición

Al activar Funnel, cualquier persona en Internet puede llegar a:

- `/.well-known/*` y `/register` (descubrimiento y registro, con rate limit),
- `/authorize` y `/token` (login de usuario único y emisión de tokens),
- `/mcp` (exige bearer token válido).

No hay “URL secreta”. La única barrera es la contraseña de administración, el hash bcrypt, PKCE, los tokens de corta duración y el rate limit. Usá una contraseña única y larga, y no reutilices la del dashboard ni la de otros servicios.

## Checklist de seguridad antes de publicar

- [ ] `P40LA_MCP_REMOTE_WRITES=false` salvo aprobación explícita del usuario.
- [ ] `P40LA_MCP_ALLOW_WRITES` también en `false` si el proceso no necesita escribir por stdio.
- [ ] Contraseña de administración única, generada y hasheada con `p40la-mcp -hash-password`.
- [ ] `P40LA_MCP_PUBLIC_URL` con `https`, sin path, query ni fragment.
- [ ] Funnel apunta solo a `127.0.0.1:8090`; nunca a `8088` ni `8091`.
- [ ] Volumen `/data` con permisos `0700` y `oauth.json`/`audit.jsonl` en `0600`.
- [ ] `P40LA_MCP_AUDIT_MAX_MB` configurado según el espacio del nodo.
- [ ] Auditoría revisada después de las primeras llamadas.
- [ ] Tokens revocados al cambiar de dispositivo o dar de baja el conector.
- [ ] Ningún dato real (tailnet, IPs, tokens, hashes) en el repo, issues, PRs ni capturas.

## Troubleshooting

| Síntoma | Causa probable | Acción |
|---|---|---|
| `tailscale funnel` falla por permisos | Falta el atributo `funnel` en `nodeAttrs` | Agregarlo en la política de la tailnet |
| Falla la emisión del certificado | HTTPS Certificates o MagicDNS deshabilitados | Activarlos en la consola de administración |
| Funnel activo pero Claude no llega | El proceso MCP no escucha en 8090 | `curl http://127.0.0.1:8090/.well-known/oauth-protected-resource` en el nodo |
| `502` desde Internet | Funnel proxya a un puerto cerrado | Verificar `P40LA_MCP_LISTEN` y que el contenedor esté corriendo |
| El dashboard dejó de verse en la tailnet | Funnel reemplazó el handler `/` de 443 | Restaurar `tailscale serve` o usar un nodo dedicado |
| Discovery responde con otro hostname | `P40LA_MCP_PUBLIC_URL` no coincide con el FQDN | Corregir la variable y reiniciar |
| El login OAuth funciona dentro de la tailnet pero no fuera | Funnel no está publicando, aunque Serve sí responda | Verificar desde datos móviles y revisar `funnel status` |

## Referencias

- [Servidor HTTP remoto](http.md) · [OAuth 2.1](oauth.md) · [Operación](operations.md) · [Seguridad](security.md)
- `deploy/tailscale/serve.json` y `deploy/tailscale/serve-funnel.example.json` en el repo.
- Documentación de Tailscale: *Funnel*, *Serve* y *HTTPS certificates*.
