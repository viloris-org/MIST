# MIST

[English](../../README.md) · [中文](README.zh.md) · [日本語](README.ja.md) · Español · [Русский](README.ru.md)

MIST es un transporte de superposición privada compacto basado en TLS para conectividad segura y confiable en implementaciones empresariales, comerciales y autogestionadas que cumplen con las normativas.

El proyecto ofrece una superficie operativa reducida, rendimiento predecible y configuración clara para equipos que necesitan acceso interno, tunelización de infraestructura, operaciones remotas y conectividad de servicios en entornos de confianza.

La implementación actual proporciona:

- Transporte cliente/servidor basado en TLS
- Multiplexación de flujos sobre una única sesión autenticada
- Acceso a aplicaciones locales a través de un escucha SOCKS5/HTTP del lado del cliente
- Biblioteca cliente multiplataforma (`mistclient`) para Android, iOS, OpenWRT y uso embebido
- Soporte configurable de relleno de paquetes
- Modos de certificado autofirmado, ACME y personalizado
- Instalación opcional de systemd a través del instalador incluido
- Integridad de tramas HMAC y autenticación de desafío-respuesta (protocolo v3)

## Instalación del Servidor en Una Línea

```bash
curl -fsSL https://mist.viloris.org/install-server.sh | bash
```

El script detecta su arquitectura Linux, descarga el binario más reciente y lo instala en `/usr/local/bin`.

Para una configuración interactiva con servicio systemd:

```bash
curl -fsSL https://mist.viloris.org/install.sh | bash
```

Los binarios precompilados están disponibles en la página de [Releases](https://github.com/viloris-org/MIST/releases).

## Compilación

```bash
go build ./cmd/mist-server
go build ./cmd/mist-client
```

## Biblioteca

El paquete `mistclient/` proporciona una biblioteca cliente multiplataforma:

```go
import "mist/mistclient"

opts := mistclient.Options{
    ServerAddr: "example.com:8443",
    Password:   "your-password",
    Logger:     myLogger, // implementa mistclient.Logger
}
client, _ := mistclient.NewClient(opts)
defer client.Close()

conn, _ := client.DialStream(ctx, destination)
```

La interfaz `Logger` permite que cada plataforma inyecte su propio registro — `android.util.Log` en Android, `os_log` en iOS, `logrus` en CLI. Consulte `../../mistclient/options.go` para la configuración completa.

## Uso Manual del Servidor

```bash
./mist-server -l 0.0.0.0:8443 -p "your-password"
```

`-l` establece la dirección de escucha. `-p` establece la contraseña compartida.

### Modos de Certificado

Autofirmado:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type self-signed \
  -cert-name 203.0.113.10
```

Si se omite `-cert-name`, el servidor lo deriva de la dirección de escucha. `0.0.0.0` se reemplaza por `127.0.0.1`.

ACME:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type acme \
  -cert-name example.com \
  -acme-http :80 \
  -acme-cache ./cert-cache
```

Personalizado:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type custom \
  -cert-file /path/to/cert.pem \
  -key-file /path/to/key.pem
```

## Uso Manual del Cliente

```bash
./mist-client -l 127.0.0.1:1080 -s example.com:8443 -p "your-password"
```

Con fijación de huella digital de certificado autofirmado:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -p "your-password" \
  -tls-cert-sha256 "server-certificate-sha256"
```

Con SNI explícito al conectarse por IP:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -sni example.com \
  -p "your-password"
```

El cliente también acepta URLs `mist://`:

```bash
./mist-client -l 127.0.0.1:1080 -s "mist://password@example.com:8443?sni=example.com"
```

## Notas de Ejecución

- Establezca `LOG_LEVEL=debug` para registros detallados.
- Establezca `TLS_KEY_LOG=/path/to/keylog.txt` solo en el cliente para depuración TLS.
- Mantenga contraseñas, claves privadas, scripts de inicio y caché de certificados fuera del control de versiones.
- Use herramientas externas de gestión de secretos y ciclo de vida de certificados para producción.

## Cumplimiento y Legal

MIST es una herramienta de transporte de red de propósito general. No recolecta telemetría, no realiza llamadas a servidores del proyecto ni incluye ningún mecanismo de puerta trasera o evasión. Como cualquier software de red, puede utilizarse tanto para fines legítimos como ilegítimos. Los autores y colaboradores proporcionan este software "tal cual" para uso exclusivamente autorizado.

### Uso Autorizado

Este software está destinado únicamente para fines legítimos y autorizados, incluyendo:

- Acceso a infraestructura interna y redes de superposición privadas
- Administración remota de sistemas y flujos de trabajo DevOps
- Conectividad segura en entornos comerciales, empresariales y autogestionados
- Uso educativo, investigación de seguridad en contextos autorizados (por ejemplo, CTF, entornos de laboratorio) y autohospedaje personal

No puede utilizar MIST para ningún propósito que viole las leyes, regulaciones o derechos de terceros aplicables. Si no está seguro de si su caso de uso está autorizado, consulte a un asesor legal calificado antes de implementar.

### Responsabilidades del Operador

Implementar MIST en cualquier entorno lo convierte en el operador de esa implementación. Los operadores son los únicos responsables de:

- Asegurar que la implementación cumpla con todas las leyes y regulaciones locales, nacionales e internacionales aplicables
- Obtener las autorizaciones, licencias o permisos necesarios antes de operar infraestructura de tunelización cifrada
- Restringir el acceso a usuarios y sistemas aprobados mediante autenticación, reglas de firewall y controles de acceso adecuados
- Proteger credenciales, claves privadas, registros, material de certificados y configuración contra accesos no autorizados
- Monitorear el estado del servicio, la capacidad y la seguridad según los requisitos operativos organizacionales
- Mantener un proceso de actualización y reversión para entornos de producción
- Cumplir con las obligaciones de protección de datos y privacidad, incluidas aquellas relacionadas con el tráfico de usuarios que pueda transitar por el servidor
- Mantener registros de implementación precisos alineados con las políticas de gestión de cambios y control de acceso

### Sin Responsabilidad

EL SOFTWARE SE PROPORCIONA "TAL CUAL", SIN GARANTÍA DE NINGÚN TIPO, EXPRESA O IMPLÍCITA, INCLUYENDO PERO NO LIMITADO A LAS GARANTÍAS DE COMERCIABILIDAD, IDONEIDAD PARA UN PROPÓSITO PARTICULAR Y NO INFRACCIÓN. EN NINGÚN CASO LOS AUTORES O TITULARES DE DERECHOS DE AUTOR SERÁN RESPONSABLES POR NINGUNA RECLAMACIÓN, DAÑO U OTRA RESPONSABILIDAD, YA SEA EN UNA ACCIÓN DE CONTRATO, AGRAVIO O DE OTRA FORMA, QUE SURJA DE O EN CONEXIÓN CON EL SOFTWARE O EL USO U OTRO TIPO DE ACCIONES EN EL SOFTWARE.

Los autores y colaboradores no asumen ninguna responsabilidad por:

- El mal uso del software por parte de operadores o usuarios finales
- Daños resultantes de una configuración incorrecta, prácticas de seguridad inadecuadas o el incumplimiento de las mejores prácticas operativas
- Violaciones de leyes, regulaciones o derechos de terceros que surjan de cualquier implementación o uso del software
- El tráfico transmitido a través de túneles MIST, incluyendo cualquier contenido ilegal o no autorizado
- Incidentes de seguridad resultantes de que el operador no aplique actualizaciones, gestione credenciales o asegure la infraestructura

### Jurisdicción y Exportación

MIST se desarrolla y distribuye globalmente. Al usar o distribuir este software, usted declara que su uso y distribución cumplen con todas las leyes de control de exportación, regulaciones de sanciones y restricciones comerciales aplicables de su jurisdicción y de cualquier jurisdicción donde se implemente el software.

### Servicios de Terceros

Si implementa MIST en infraestructura proporcionada por terceros (proveedores de nube, hosts VPS, servicios CDN, registradores de dominios, autoridades de certificación, etc.), usted es responsable de cumplir con los términos de servicio y políticas de uso aceptable de dichos proveedores. Los autores no hacen ninguna declaración de que el uso de MIST esté permitido bajo los términos de ningún proveedor específico.

### Informes

Para informar una vulnerabilidad de seguridad, abra un aviso de seguridad privado en el repositorio de GitHub o envíe un correo electrónico a <connect@viloris.org>. Para otras inquietudes, contacte a los mantenedores a través de los canales oficiales del proyecto.

---

Protocolo v3 con integridad de tramas HMAC y autenticación de desafío-respuesta. Compatible con `mist/0.0.2` y versiones posteriores.
