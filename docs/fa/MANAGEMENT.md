<div dir="rtl">

# مدیریت XNode روی سرور

[English](../MANAGEMENT.md) | [فارسی](MANAGEMENT.md)

از نسخهٔ `v1.1.0` به بعد، دستور دوزبانهٔ `xnode` در مسیر
`/usr/local/bin/xnode` نصب می‌شود. برای بازکردن منوی تعاملی اجرا کنید:

```bash
sudo xnode
```

این منو سرویس `xnode-agent.service` را مدیریت می‌کند. Xray پردازش فرزند ایجنت
است؛ بنابراین restart سرویس، ایجنت و Xray مدیریت‌شده را با هم و به‌شکل امن
راه‌اندازی مجدد می‌کند. عمداً گزینهٔ جداگانه‌ای برای اجرای Xray خارج از کنترل
ایجنت وجود ندارد.

## دستورها

```text
xnode status                  نمایش وضعیت systemd، ایجنت و Xray
xnode start                   فعال‌سازی و شروع سرویس
xnode stop                    توقف سرویس
xnode restart                 راه‌اندازی مجدد ایجنت و Xray
xnode logs [--follow]         لاگ ایجنت
xnode xray-logs [--follow]    access log هستهٔ Xray
xnode settings                نمایش تنظیمات با مخفی‌کردن کامل token
xnode configure               ویرایش امن و تعاملی اتصال پنل
xnode test-panel              تست درخواست احراز هویت‌شده به پنل
xnode doctor                  عیب‌یابی کامل محلی و پنل
xnode update [vX.Y.Z]         ارتقا، نصب نسخهٔ مشخص یا rollback
xnode backup                  پشتیبان‌گیری از config و state
xnode backups                 فهرست فایل‌های پشتیبان
xnode uninstall [--purge]     حذف با حفظ اطلاعات یا حذف کامل
xnode version                 نمایش نسخه‌های نصب‌شده
```

دستورهای حساس در صورت وجود `sudo` به‌طور خودکار با دسترسی root اجرا می‌شوند.
فایل config با permission برابر `0600` باقی می‌ماند و دستور `settings` هیچ‌وقت
token را چاپ نمی‌کند. `configure` شناسهٔ نود، URL امن پنل، token و فاصلهٔ sync
را پیش از نصب فایل بررسی می‌کند. اگر سرویس با تنظیمات جدید health check را پاس
نکند، config قبلی خودکار برگردانده می‌شود.

`test-panel` همان درخواست احراز هویت‌شدهٔ desired-state ایجنت را اجرا می‌کند و
بدون نمایش token، نتیجهٔ HTTP را گزارش می‌دهد. `doctor` علاوه بر آن، وجود
binaryها، اعتبار JSON، permission، systemd و endpointهای health و readiness را
بررسی می‌کند.

## ارتقا و بازگشت نسخه

ارتقا به آخرین release با حفظ config فعلی:

```bash
sudo xnode update
```

نصب یا بازگشت به یک release مشخص:

```bash
sudo xnode update v1.0.2
```

installer اصلی همچنان archive را با `SHA256SUMS` تطبیق می‌دهد، backup موقت
می‌گیرد و در صورت شکست نصب rollback می‌کند.

## پشتیبان‌گیری و حذف

دستور `sudo xnode backup` یک archive فقط‌خواندنی برای root در مسیر
`/var/backups/xnode` می‌سازد. حذف عادی مسیرهای `/etc/xnode`، `/var/lib/xnode` و
`/var/log/xnode` را نگه می‌دارد. گزینهٔ `--purge` همهٔ آن‌ها را برای همیشه حذف
می‌کند و نیازمند واردکردن عبارت تأیید `PURGE` است.

## نکات امنیتی

- token واقعی نود را در آرگومان خط فرمان، issue یا screenshot قرار ندهید؛
- status ایجنت و Xray API را روی loopback نگه دارید؛
- منو پورت firewall را باز نمی‌کند و فایل تولیدشدهٔ Xray را دستی تغییر نمی‌دهد؛
- inbound، outbound، DNS و routing را از desired state پنل مدیریت کنید.

</div>
