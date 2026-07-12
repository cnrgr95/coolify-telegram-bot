# Kurulum

## Gereksinimler

- Go 1.25
- TDLib uyumlu `libtdjson`
- Coolify API erişimi
- Telegram bot tokenı ve API bilgileri
- Kalıcı Docker volume

## Docker

Image oluşturun:

```sh
docker build -t flpanel-bot .
```

Container içinde `/app/data` yolunu kalıcı volume'a bağlayın ve `.env.example` içindeki zorunlu değişkenleri tanımlayın.

## Coolify

- Build pack: Dockerfile
- Port: `8080`
- Persistent Storage hedefi: `/app/data`
- Health endpoint: `/health`
- Secret değerleri yalnız Coolify environment ekranından girin.
