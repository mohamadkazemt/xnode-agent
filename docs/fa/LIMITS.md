<div dir="rtl">

# اعمال محدودیت‌ها و مدیریت session

[English](../LIMITS.md) | [فارسی](LIMITS.md)

## هویت کاربر

ایجنت برای هر credential مدیریت‌شده یک email قطعی Xray می‌سازد:

```text
u:<user_id>|i:<inbound_id>
```

همین هویت برای آمار ترافیک، IP آنلاین، routing و policy سخت‌گیرانهٔ dispatcher استفاده می‌شود.

## محدودیت حجم

پنل هم سقف حجم و هم مصرف تجمیعی ثبت‌شده را می‌فرستد:

```json
{"traffic_bytes":536870912000,"traffic_used_bytes":420000000000}
```

ایجنت پیش از ارزیابی policy، batchهای صف‌شده را retry می‌کند و deltaهای محلی هنوز commit‌نشده را در نظر می‌گیرد. کاربر در این حالت block می‌شود:

```text
traffic_used_bytes + pending_unsent_bytes >= traffic_bytes
```

با هستهٔ patch‌شده، session موجود در read/write بعدی block را می‌بیند و بسته می‌شود. بدون patch، حذف credential فقط ورود جدید را می‌بندد و اتصال قبلی ممکن است تا پایان طبیعی باقی بماند.

## انقضا

`expires_at` زمان Unix بر حسب ثانیه است. در آن زمان یا بعد از آن کاربر غیرفعال می‌شود و همان رفتار قطع session محدودیت حجم اعمال می‌شود. مقدار صفر یعنی بدون انقضا.

## محدودیت IP

هستهٔ patch‌شده `ip_limit` را هنگام ورود dispatcher اعمال می‌کند. IPهای source فعال برای هویت کاربر ref-count می‌شوند. وقتی سقف پر باشد، IP متمایز جدید فوراً رد می‌شود؛ چند اتصال از IP ازقبل‌پذیرفته‌شده slot دیگری مصرف نمی‌کند.

ایجنت IPهای آنلاین بومی Xray را نیز برای گزارش می‌خواند و در صورت نیاز از access log استفاده می‌کند. اگر پشت CDN یا reverse proxy، Xray فقط IP واسط را می‌بیند، برای inbound مقدار `ip_limit_mode: "off"` قرار دهید.

## محدودیت دستگاه

از IP نمی‌توان دستگاه فیزیکی را دقیق تشخیص داد. برای هر دستگاه credential جدا بسازید و به آن‌ها `account_id` مشترک بدهید:

```json
{"id":"device-a","account_id":"25","limits":{"device_limit":2}}
```

ایجنت حداکثر N credential فعال نخست همان account را در هر inbound می‌پذیرد. پنل نیز هنگام ساخت credential باید همین سقف را بررسی کند.

## محدودیت سرعت

مقادیر بر حسب **بایت‌برثانیه** هستند:

```json
{"upload_bps":2500000,"download_bps":12500000}
```

این نمونه تقریباً ۲۰ مگابیت upload و ۱۰۰ مگابیت download است. یک bucket مشترک برای هر کاربر و جهت استفاده می‌شود تا بازکردن اتصال‌های بیشتر سقف را چندبرابر نکند.

## محدودیت اتصال

`connection_limit` تعداد اتصال منطقی dispatcher برای کاربر احرازشده را محدود می‌کند. اتصال‌ها حتی وقتی limit وجود ندارد شمارش می‌شوند تا کاهش بعدی limit روی sessionهای موجود نیز قابل اعمال باشد.

## Drain نود

در حالت `draining`، config فعلی باقی می‌ماند ولی هسته ورود **جدید** کاربر احرازشده را رد می‌کند. اتصال‌های ازقبل‌پذیرفته‌شده صرفاً به‌دلیل شروع drain قطع نمی‌شوند و فرصت پایان دارند. وقتی تعداد کاربران آنلاین به `drain_target_online` برسد، heartbeat مقدار `drain_ready:true` می‌فرستد.

این رفتار با `maintenance` و `disabled` متفاوت است؛ آن دو Xray را متوقف می‌کنند.

## Suspend و Resume

برای تعلیق credential مقدار `enabled:false` بفرستید. کاربر از authentication حذف و در policy block می‌شود. با `enabled:true` دوباره فعال می‌شود.

اگر کاربر کاملاً از desired state حذف شود، یک tombstone مسدود کوتاه‌مدت در limits file باقی می‌ماند تا session قدیمی پس از حذف runtime باز نماند.

## قطع sessionها بدون تعلیق

عدد `session_generation` را در پنل افزایش دهید و کاربر را enabled نگه دارید. sessionهای جدید generation تازه را می‌گیرند و اتصال‌های generation قبلی در عملیات بعدی داده بسته می‌شوند. این قابلیت برای دکمهٔ «قطع همهٔ اتصال‌ها» بدون تعویض UUID/password مناسب است.

## Limiter خارجی اختیاری

`strict_limit_backend_url` یک extension hook است. اگر تنظیم شود، ایجنت محدودیت‌ها را mirror می‌کند:

```text
PUT    /v1/limits/{node}/{inbound}/{user}
DELETE /v1/limits/{node}/{inbound}/{user}
```

با Xray patch‌شدهٔ همراه release به این backend نیازی نیست.

## WireGuard

ایجنت برای peerهای WireGuard نیز همان `email` و `level` قطعی را وارد می‌کند. Xray peer را از source address مجاز پیدا می‌کند و هویت را به dispatcher می‌دهد؛ بنابراین آمار کاربر، IP آنلاین، routing و limiter برای WireGuard نیز کار می‌کنند. تغییر عضویت peer با جایگزینی inbound انجام می‌شود، چون CLI فعلی Xray کاربران WireGuard را برای `adu` استخراج نمی‌کند.

</div>

