# Migration Planı

## Faz 0 — Stabilizasyon

- Mevcut Go sürümünü çalışır tut
- `/app/data` volume ve yedeklemeyi doğrula
- Türkçe encoding, yetki ve callback testlerini tamamla
- Kritik işlemlere onay ve audit ekle

## Faz 1 — Ortak Backend

- TypeScript monorepo oluştur
- Domain modelleri ve ortak işlem servislerini tanımla
- Coolify adapter contract ve integration testleri ekle
- Telegram ve web işlemlerini aynı API'ye geçir

## Faz 2 — Kalıcı Veri ve Güvenlik

- PostgreSQL şeması ve JSON migration aracı
- Redis rate limit ve job coordination
- Güvenli session, CSRF ve brute-force koruması
- Değiştirilemez audit olay zinciri

## Faz 3 — Monitoring ve Uyarılar

- Kimlik doğrulamalı agent
- Ham, 5 dakikalık ve saatlik metric tabloları
- Retention ve aggregation worker'ları
- Cooldown ve recovery destekli alert engine

## Faz 4 — Arayüz Eşitliği

- Responsive web uygulaması
- Telegram komut ve inline menü parity'si
- SSE/WebSocket gerçek zamanlı olaylar
- Grafik, filtre, arama ve sayfalama

Her faz bağımsız deploy edilmeli ve geri dönüş planı içermelidir. Go uygulaması, TypeScript backend doğrulanana kadar kaldırılmayacaktır.
