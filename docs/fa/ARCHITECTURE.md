<div dir="rtl">

# معماری نسخه ۱.۰

[English](../ARCHITECTURE.md) | [فارسی](ARCHITECTURE.md)

```text
پنل / Control Plane
کاربران، نودها، تخصیص، گروه‌ها، failover و load balancing
                         |
                   HTTPS + token
                         v
xnode-agent
desired-state | policy | traffic spool | session | health/recovery
                         |
              API محلی + فایل‌های atomic
                         v
Xray v26.7.28 + لایهٔ محدودکنندهٔ xnode
protocols | transports | stats | routing | strict limits
                         |
                      اینترنت
```

## مسئولیت هر بخش

پنل منبع حقیقت سراسری است: کاربران، پلن‌ها، credentialها، مصرف تجمیعی، تخصیص نود، گروه‌ها و زمان‌بندی بین نودها را مدیریت می‌کند. ایجنت فقط یک سرور را به وضعیت مطلوب می‌رساند. Xray مسیر واقعی انتقال داده است.

این جداسازی باعث می‌شود سرورها قابل جایگزینی باشند و یک نود نقش دیتابیس سراسری نگیرد.

## چرخهٔ همگرایی

```text
retry رویدادهای ترافیک روی دیسک
  -> خواندن و reset شمارنده‌های Xray
  -> spool و ارسال batch جدید
  -> دریافت IPهای آنلاین
  -> GET desired state
  -> محاسبهٔ mode و drain
  -> ارزیابی حجم، انقضا، IP و دستگاه
  -> نوشتن atomic policy هسته
  -> ساخت config کامل Xray
  -> xray run -test
  -> تغییر hot یا restart اعتبارسنجی‌شده
  -> health check / recovery / rollback
  -> POST sessions و heartbeat
  -> ذخیرهٔ state واقعاً اعمال‌شده
```

state ذخیره‌شده وضعیت مؤثر اعمال‌شده است و blockهای موقت policy را نیز شامل می‌شود. وقتی پنل دوباره کاربر را مجاز کند، چرخهٔ بعدی او را برمی‌گرداند.

## تغییرات زمان اجرا

برای inboundها و کاربران VLESS/VMess/Trojan/Shadowsocks در صورت امکان عملیات hot انجام می‌شود. تغییر peerهای WireGuard باعث جایگزینی همان inbound می‌شود. تغییرات سراسری outbound، routing، DNS یا policy level با restart اعتبارسنجی‌شده اعمال می‌شوند.

اگر یک عملیات hot شکست بخورد، ایجنت config کامل را اعمال می‌کند. اگر شروع process نیز شکست بخورد، آخرین config سالم ذخیره‌شده امتحان می‌شود.

## اعمال محدودیت در مسیر داده

ایجنت یک نقشهٔ JSON اتمیک با هویت آماری کاربر می‌نویسد. dispatcher patch‌شده:

1. کاربر احرازشده را می‌شناسد؛
2. اتصال‌های منطقی او را می‌شمارد؛
3. block و اجازهٔ اتصال را بررسی می‌کند؛
4. مسیر upload/download را کنترل می‌کند؛
5. policy را بدون restart دوباره می‌خواند؛
6. یک rate bucket مشترک برای هر کاربر و جهت اعمال می‌کند؛
7. sessionهای block‌شده، حذف‌شده، عبورکرده از limit یا generation قدیمی را می‌بندد.

## Failover و Load Balancing

یک نود دید معتبری از همهٔ نودها ندارد، پس تصمیم failover سراسری باید در پنل انجام شود. heartbeat اطلاعات health، region، group، tags، weight، mode، drain، سرعت شبکه و threshold را برای scheduler پنل می‌فرستد. داخل یک process همچنان می‌توان از routing و balancer بومی Xray استفاده کرد.

</div>

