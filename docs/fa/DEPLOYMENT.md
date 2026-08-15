<div dir="rtl">

# نصب و استقرار

[English](../DEPLOYMENT.md) | [فارسی](DEPLOYMENT.md)

## پیش‌نیازها

- Linux روی `amd64` یا `arm64`؛
- systemd برای نصب واقعی؛ گزینهٔ `--no-start` فقط برای CI و ساخت image است؛
- `curl` یا `wget`، `python3`، `tar` و `sha256sum` یا `shasum`.

Debian و Ubuntu پیشنهاد می‌شوند، اما installer از package manager خاصی استفاده نمی‌کند.

## نصب آخرین release

نصب تعاملی:

```bash
sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh)"
```

نصب غیرتعاملی:

```bash
curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh | \
  sudo bash -s -- \
  --node-id node-12 \
  --panel-url https://panel.example.com \
  --panel-token 'UNIQUE_NODE_TOKEN'
```

فایل config آماده و pin کردن نسخه:

```bash
sudo bash scripts/install.sh --config ./agent.json --version v1.1.0
```

گزینه‌ها:

- `--node-id`: شناسهٔ یکتای نود؛
- `--panel-url`: آدرس HTTPS پنل؛
- `--panel-token`: token اختصاصی نود؛
- `--config`: فایل JSON آماده؛
- `--version`: نسخهٔ مشخص؛ اگر حذف شود latest نصب می‌شود؛
- `--force`: اجازهٔ بازنویسی config قبلی؛
- `--no-start`: نصب فایل‌ها بدون اجرای systemd، مخصوص CI/image.

config موجود بدون `--force` تغییر نمی‌کند. token واقعی را در repository یا shell history قرار ندهید؛ برای automation از فایل فقط‌خواندنی root استفاده کنید.

installer معماری را تشخیص می‌دهد، archive رسمی را دانلود می‌کند، SHA-256 را با `SHA256SUMS` تطبیق می‌دهد و این مسیرها را می‌سازد:

```text
/usr/local/bin/xnode-agent
/usr/local/bin/xray
/usr/local/bin/xnode                   (منوی مدیریت در v1.1.0+)
/usr/local/lib/xnode/uninstall.sh
/usr/local/lib/xnode/VERSION
/etc/xnode/agent.json             (0600)
/etc/systemd/system/xnode-agent.service
/etc/logrotate.d/xnode
/var/lib/xnode/traffic-spool/
/var/log/xnode/
```

دایرکتوری‌های config و data با mode `0700` ساخته می‌شوند. شکست در نصب، شروع سرویس یا health check باعث rollback فایل‌های مدیریت‌شده می‌شود.

بعد از نصب، منوی دوزبانه را اجرا کنید:

```bash
sudo xnode
```

دستورها، ویرایش امن اتصال، عیب‌یابی، ارتقا، پشتیبان‌گیری و حذف در
[راهنمای مدیریت سرور](MANAGEMENT.md) توضیح داده شده‌اند.

## بررسی نصب

```bash
systemctl status xnode-agent
journalctl -u xnode-agent -n 100 --no-pager
curl -fsS http://127.0.0.1:19090/healthz
curl -fsS http://127.0.0.1:19090/readyz
curl -fsS http://127.0.0.1:19090/status
```

## ارتقا یا بازگشت نسخه

installer را دوباره با `--force` اجرا کنید. در طول عملیات از فایل‌های قبلی backup موقت گرفته می‌شود. برای بازگشت، نسخهٔ قبلی را با `--version vX.Y.Z` مشخص کنید.

## حذف

حذف binary و سرویس با حفظ config، state و log:

```bash
sudo bash scripts/uninstall.sh
```

حذف کامل اطلاعات نود:

```bash
sudo bash scripts/uninstall.sh --purge
```

## Firewall

فقط پورت inboundهای موردنیاز Xray را باز کنید. Xray API روی `127.0.0.1:10085` و status ایجنت روی `127.0.0.1:19090` نباید از اینترنت قابل دسترسی باشند.

</div>

