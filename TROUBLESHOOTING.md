# Sorun Giderme

## Kullanıcılar redeploy sonrası kayboluyor

Coolify Persistent Storage hedefinin `/app/data` olduğunu doğrulayın.

## Sistem metrikleri N/A

Sentinel veya monitoring agent erişilebilir değil ya da ilgili endpoint metrik sağlamıyor. Uygulama sahte değer üretmez.

## Bot başlamıyor

`TOKEN`, `API_ID`, `API_HASH`, `API_URL` ve `API_TOKEN` değerlerini kontrol edin. TDLib dosya yolunu doğrulayın.

## Deployment build hatası

`go generate ./...` ve `CGO_ENABLED=0 go build -o bot .` komutlarını Linux hedefinde doğrulayın.
