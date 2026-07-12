# Mimari

## Bugünkü Durum

- `src/coolity`: Coolify HTTP istemcisi ve kısa süreli cache
- `src/config`: ortam değişkenleri ve Telegram rol matrisi
- `src/database`: JSON tabanlı kullanıcı ve görev deposu
- `src/scheduler`: zamanlanmış Coolify işlemleri
- `src`: Telegram adapter ve bildirimler
- `web.go`: web oturumu, HTML ve web işlem handler'ları

Telegram callback'leri ile web handler'ları Coolify istemcisini doğrudan çağırmaktadır. Bu nedenle özellik eşitliği ve audit kaydı garanti edilememektedir.

## Hedef Mimari

1. TypeScript API/backend
2. Ortak application service katmanı
3. Coolify adapter
4. Güvenli monitoring-agent adapter
5. PostgreSQL repository katmanı
6. Redis cache, rate limit ve job coordination
7. Alert, notification ve audit servisleri
8. Telegram worker adapter
9. Web frontend adapter

Telegram ve web hiçbir yönetim işlemini doğrudan Coolify'a göndermeyecektir. Her işlem ortak serviste yetkilendirilecek, audit kaydı oluşturacak ve aynı sonuç tipini döndürecektir.

## Monitoring Sınırı

Coolify API container CPU, RAM, ağ ve volume geçmişini eksiksiz sağlamaz. Bu veriler için Docker socket'i internete açmayan, sunucu içinde çalışan kimlik doğrulamalı bir agent gerekir. Agent hazır olmadan panelde sahte metrik gösterilmeyecektir.
