# Güvenlik

## Mevcut Koruma

- Web parolaları bcrypt ile hashlenir.
- Session cookie `HttpOnly`, `Secure` ve `SameSite=Lax` kullanır.
- Telegram callback'lerinde rol kontrolleri yeniden yapılır.
- Coolify HTTP istemcisinde timeout vardır.

## Açık Riskler

- Web POST işlemlerinde CSRF token bulunmuyor.
- Login rate limit ve kalıcı brute-force kilidi bulunmuyor.
- Session'lar process belleğinde; restart sonrası siliniyor.
- Coolify ve Telegram tokenları environment içinde düz metin olarak sağlanıyor.
- Yönetim işlemlerinin değiştirilemez audit kaydı bulunmuyor.
- JSON deposu eşzamanlı çok instance çalışmasına uygun değil.
- Monitoring agent imza/mTLS tasarımı henüz uygulanmadı.

## Zorunlu Production Kuralları

- Tokenları repository veya dokümana yazmayın.
- Coolify API tokenına yalnız gerekli izinleri verin.
- `/app/data` için şifreli yedek alın.
- Docker socket'i public ağa açmayın.
- Reverse proxy üzerinde TLS ve request limitleri kullanın.
- TypeScript migration tamamlanmadan çoklu replica çalıştırmayın.

Güvenlik açığı bildirimlerinde token, parola veya kişisel veri paylaşmayın; etkilenen sürüm ve yeniden üretim adımlarını özel kanaldan iletin.
