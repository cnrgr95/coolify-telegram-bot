# Mevcut Durum ve Özellik Eşitliği

| Özellik | Telegram | Web | Ortak Servis | Durum |
|---|---:|---:|---:|---|
| Uygulama listesi | Var | Var | Hayır | Refactor gerekli |
| Servis listesi | Var | Uygulamalar içinde | Hayır | Ayrı görünüm gerekli |
| Veritabanı listesi | Var | Var | Coolify client | Kısmi |
| Sunucu listesi | Sistem özeti | Var | Coolify client | Detay gerekli |
| Deploy / redeploy | Var | Var | Hayır | Audit ve ortak servis gerekli |
| Restart / stop | Var | Var | Hayır | Onay standardı gerekli |
| Silme | Var | Yok | Hayır | Tehlikeli işlem tasarımı gerekli |
| Uygulama logları | Var | Var | Hayır | Ortak servis gerekli |
| Zamanlanmış görevler | Var | Yok | Scheduler | Web parity eksik |
| Telegram kullanıcıları | Var | Var | JSON repository | Mevcut |
| Web kullanıcıları | Var | Var | JSON repository | Mevcut |
| Durum bildirimi | Var | Yok | Notification | Web olay merkezi eksik |
| CPU/RAM anlık | Kısmi | Kısmi | Sentinel | Kaynağa göre detay yok |
| Metric geçmişi | Yok | Yok | Yok | Agent ve PostgreSQL gerekli |
| Alert kuralları | Yok | Yok | Yok | Alert service gerekli |
| Audit log | Yok | Yok | Yok | Kritik güvenlik açığı |
| Deployment geçmişi | Yok | Yok | Yok | Coolify adapter genişletilmeli |

## Kritik Teknik Borç

1. Telegram ve web yönetim işlemleri aynı servis fonksiyonunu kullanmıyor.
2. JSON veri deposu kullanıcı, görev ve metric ölçeği için yeterli değil.
3. Web POST formlarında CSRF token henüz yok.
4. Monitoring geçmişi ve container agent bulunmuyor.
5. Audit ve alert domain modelleri bulunmuyor.
6. Mevcut UI server-side tek HTML template; hedef responsive frontend için ayrıştırılmalı.

Bu tablo her migration fazında güncellenecek kabul kriteridir.
