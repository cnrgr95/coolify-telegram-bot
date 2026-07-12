# FL Panel Coolify Yönetimi

Coolify kaynaklarını Telegram ve web panelinden yöneten mevcut Go uygulamasıdır. Uygulamalar, servisler, veritabanları, kullanıcılar ve zamanlanmış işlemler tek process içinde çalışır.

## Mevcut Özellikler

- Coolify uygulama, servis, sunucu ve veritabanı envanteri
- Başlatma, durdurma, yeniden başlatma, deploy ve redeploy
- Uygulama logları
- Telegram ve web kullanıcıları için rol kontrolü
- Kalıcı JSON veri deposunda kullanıcılar ve zamanlanmış görevler
- Telegram hızlı menüsü ve web paneli
- Kaynak durum değişikliği ve zamanlanmış görev bildirimleri

## Yerel Geliştirme

1. `.env.example` dosyasını `.env` olarak kopyalayın.
2. Coolify ve Telegram değerlerini doldurun.
3. `go mod download` çalıştırın.
4. `go generate ./...` çalıştırın.
5. `go run .` ile başlatın.

## Production

Mevcut sürüm Dockerfile ile tek container olarak çalışır. `/app/data` dizini mutlaka kalıcı volume olmalıdır. Coolify hedef yolu `/app/data` olarak ayarlanmazsa kullanıcılar ve görevler redeploy sonrasında kaybolur.

Hedef çok servisli mimari ve migration sırası için `ARCHITECTURE.md` ve `MIGRATION.md` dosyalarına bakın.
