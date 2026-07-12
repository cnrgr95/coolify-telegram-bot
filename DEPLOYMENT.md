# Deployment

1. Testleri çalıştırın.
2. Docker image'ı oluşturun.
3. `/app/data` volume'unun bağlı olduğunu doğrulayın.
4. Coolify secret değişkenlerini doğrulayın.
5. Deployment başlatın.
6. `/health`, Telegram `/start` ve web login smoke testlerini yapın.
7. Kullanıcıların ve zamanlanmış görevlerin korunduğunu doğrulayın.

Başarısız migration durumunda önceki image'a rollback yapılmalı; persistent volume silinmemelidir.
