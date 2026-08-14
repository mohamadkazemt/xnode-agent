<div dir="rtl">

# راهنمای فارسی xnode-agent نسخه ۱.۰

[English](README.md) | [فارسی](README.fa.md)

`xnode-agent` برنامه‌ای است که روی هر سرور لینوکسی کنار هستهٔ Xray اجرا می‌شود. پنل شما منبع اصلی کاربران و تنظیمات است؛ ایجنت تنظیمات را از پنل می‌گیرد، آن‌ها را بررسی و روی Xray اعمال می‌کند، آمار مصرف و کاربران آنلاین را پس می‌فرستد و در صورت خرابی Xray آن را بازیابی می‌کند.

## قابلیت‌ها

- نصب خودکار ایجنت و هستهٔ patch‌شدهٔ Xray روی `amd64` و `arm64`؛
- پشتیبانی از چند inbound مستقل روی یک سرور؛
- مدیریت VLESS، VMess، Trojan، Shadowsocks و peerهای WireGuard؛
- عبور مستقیم تنظیمات بومی Xray مانند transport، Reality، routing، DNS و outbound؛
- افزودن/حذف کاربران و inboundها بدون restart در موارد پشتیبانی‌شده؛
- اعتبارسنجی تنظیمات با `xray run -test` پیش از فعال‌سازی؛
- محدودیت حجم، تاریخ انقضا، سرعت، IP، دستگاه و تعداد اتصال؛
- قطع sessionهای قبلی با `session_generation` بدون تغییر UUID یا رمز؛
- ثبت مصرف به تفکیک کاربر و inbound؛
- صف پایدار ترافیک برای جلوگیری از گم‌شدن آمار هنگام قطع پنل؛
- گزارش کاربران آنلاین، IPها، CPU، RAM، Load و ترافیک شبکه؛
- health check محلی، restart خودکار و بازگشت به آخرین تنظیمات سالم؛
- حالت‌های `active`، `draining`، `maintenance` و `disabled`؛
- systemd، logrotate، permissionهای امن، health check نصب و rollback خودکار.

## ایجنت چگونه کار می‌کند؟

به‌طور پیش‌فرض هر ۱۵ ثانیه یک چرخه اجرا می‌شود:

1. ترافیک‌های ارسال‌نشدهٔ قبلی را دوباره به پنل می‌فرستد.
2. شمارنده‌های مصرف Xray را می‌خواند و نتیجه را روی دیسک صف می‌کند.
3. کاربران و IPهای آنلاین را جمع‌آوری می‌کند.
4. desired state جدید را از پنل می‌گیرد.
5. محدودیت‌ها و وضعیت کاربران را محاسبه می‌کند.
6. تنظیمات جدید را اعتبارسنجی و روی Xray اعمال می‌کند.
7. sessionها، وضعیت سلامت و مشخصات سرور را به پنل می‌فرستد.

برای آمار نزدیک به لحظه‌ای می‌توان `sync_seconds` را روی ۵ قرار داد. مقدار ۱ ثانیه برای شروع توصیه نمی‌شود، چون تعداد درخواست‌ها و فشار دیتابیس را زیاد می‌کند. آمار حسابداری همچنان با شناسهٔ یکتای رویداد ارسال می‌شود تا دوبار محاسبه نشود.

## پیش‌نیازها

- Linux با معماری `amd64` یا `arm64`؛
- systemd برای نصب عادی؛
- دسترسی خروجی HTTPS به GitHub و پنل؛
- `curl` یا `wget`، به همراه `python3`، `tar` و `sha256sum`؛
- یک `node_id` یکتا و token جداگانه برای هر نود.

Debian و Ubuntu پیشنهاد می‌شوند، ولی installer تا جای ممکن به توزیع خاصی وابسته نیست.

## نصب یک‌خطی

برای نصب تعاملی آخرین نسخه:

```bash
sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh)"
```

installer سه مقدار می‌پرسد:

- `node_id`: شناسهٔ یکتای همین سرور در پنل؛
- `panel_url`: آدرس اصلی پنل، مانند `https://panel.example.com`؛
- `panel_token`: توکن اختصاصی همین نود.

نصب غیرتعاملی:

```bash
curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh | \
  sudo bash -s -- \
  --node-id node-12 \
  --panel-url https://panel.example.com \
  --panel-token 'UNIQUE_NODE_TOKEN'
```

برای فایل تنظیمات آماده یا نسخهٔ مشخص:

```bash
sudo bash scripts/install.sh --config ./agent.json --version v1.0.1
```

تنظیمات قبلی بدون `--force` بازنویسی نمی‌شوند. installer فایل release را دانلود و با `SHA256SUMS` بررسی می‌کند و اگر نصب یا health check شکست بخورد، فایل‌های قبلی را برمی‌گرداند.

راهنمای کامل نصب و حذف: [docs/fa/DEPLOYMENT.md](docs/fa/DEPLOYMENT.md)

## اتصال ایجنت به پنل

پنل باید چهار endpoint زیر را پیاده‌سازی کند:

```text
GET  /api/v1/nodes/{node_id}/desired-state
POST /api/v1/nodes/{node_id}/heartbeat
POST /api/v1/nodes/{node_id}/traffic
POST /api/v1/nodes/{node_id}/sessions
```

تمام درخواست‌ها این header را دارند:

```http
Authorization: Bearer <unique node token>
```

پنل باید token را با `node_id` تطبیق دهد. برای هر نود token متفاوت بسازید و token واقعی را داخل Git یا log قرار ندهید.

نمونهٔ حداقلی پاسخ desired state:

```json
{
  "version": "1",
  "enabled": true,
  "mode": "active",
  "node": {"region":"DE", "group":"default", "weight":100},
  "inbounds": [],
  "outbounds": [],
  "routing": {},
  "dns": {}
}
```

قرارداد کامل JSON و نمونهٔ کاربر/inbound در [docs/fa/PANEL_API.md](docs/fa/PANEL_API.md) آمده است.

## فایل تنظیمات ایجنت

مسیر پیش‌فرض فایل:

```text
/etc/xnode/agent.json
```

نمونهٔ ساده:

```json
{
  "node_id": "node-12",
  "panel_url": "https://panel.example.com",
  "panel_token": "UNIQUE_NODE_TOKEN",
  "sync_seconds": 5,
  "xray_binary": "/usr/local/bin/xray",
  "xray_config": "/etc/xnode/xray.json",
  "xray_api": "127.0.0.1:10085",
  "listen": "127.0.0.1:19090",
  "report_sessions": true,
  "require_patched_core": true
}
```

بعد از ویرایش:

```bash
sudo chmod 600 /etc/xnode/agent.json
sudo systemctl restart xnode-agent
```

`panel_url` باید HTTPS باشد؛ HTTP فقط برای loopback مجاز است. API داخلی Xray و health endpoint ایجنت نیز باید روی loopback باقی بمانند.

## آمار و محاسبهٔ مصرف

هویت آماری هر کاربر به شکل زیر است:

```text
u:<user_id>|i:<inbound_id>
```

بنابراین مصرف یک کاربر روی inboundهای مختلف قابل تفکیک است. هر batch ترافیک یک `event_id` یکتا دارد. پنل باید `event_id` را در همان transaction ثبت مصرف deduplicate کند؛ در غیر این صورت retry شبکه ممکن است مصرف را دو بار اضافه کند.

اگر پنل قطع شود، batchها در مسیر زیر باقی می‌مانند و بعداً دوباره ارسال می‌شوند:

```text
/var/lib/xnode/traffic-spool/
```

جزئیات: [docs/fa/TRAFFIC_DELIVERY.md](docs/fa/TRAFFIC_DELIVERY.md)

## محدودیت کاربران

پنل می‌تواند برای هر credential موارد زیر را بفرستد:

- `traffic_bytes` و `traffic_used_bytes` برای حجم؛
- `expires_at` برای انقضا؛
- `upload_bps` و `download_bps` بر حسب بایت‌برثانیه؛
- `ip_limit` برای IPهای هم‌زمان؛
- `device_limit` با credential مجزا و `account_id` مشترک؛
- `connection_limit` برای تعداد اتصال؛
- `session_generation` برای قطع sessionهای قدیمی.

راهنمای دقیق: [docs/fa/LIMITS.md](docs/fa/LIMITS.md)

## بررسی وضعیت

```bash
sudo systemctl status xnode-agent
sudo journalctl -u xnode-agent -f
curl -fsS http://127.0.0.1:19090/healthz
curl -fsS http://127.0.0.1:19090/readyz
curl -fsS http://127.0.0.1:19090/status
```

- `healthz`: خود ایجنت و اجزای ضروری سالم‌اند؛
- `readyz`: نود آمادهٔ سرویس‌دهی است؛
- `status`: جزئیات وضعیت فعلی را برمی‌گرداند.

این endpointها عمداً فقط محلی هستند. آن‌ها را روی اینترنت منتشر نکنید.

## فایل‌ها و مسیرهای مهم

```text
/usr/local/bin/xnode-agent          فایل اجرایی ایجنت
/usr/local/bin/xray                 هستهٔ patch‌شده
/etc/xnode/agent.json               تنظیمات و token
/etc/xnode/xray.json                تنظیمات تولیدشدهٔ Xray
/var/lib/xnode/state.json           state اعمال‌شده
/var/lib/xnode/limits.json          policy محدودیت‌ها
/var/lib/xnode/traffic-spool/       صف مصرف ارسال‌نشده
/var/log/xnode/xray-access.log      access log هسته
```

## حالت‌های نود

- `active`: کار عادی و همگام‌سازی کامل؛
- `draining`: اتصال جدید پذیرفته نمی‌شود، ولی اتصال‌های قبلی فرصت پایان دارند؛
- `maintenance`: گزارش وضعیت ادامه دارد ولی Xray متوقف می‌شود؛
- `disabled`: Xray متوقف و نود غیرفعال می‌ماند.

Failover و load balancing بین چند سرور باید در پنل انجام شود. ایجنت فقط وضعیت لازم مانند health، region، group، weight و drain state را گزارش می‌کند.

## ارتقا، rollback و حذف

ارتقا به آخرین نسخه:

```bash
sudo bash install.sh --config /etc/xnode/agent.json --force
```

نصب نسخهٔ مشخص:

```bash
sudo bash install.sh --config /etc/xnode/agent.json --version v1.0.1 --force
```

حذف برنامه با حفظ تنظیمات و داده‌ها:

```bash
sudo bash scripts/uninstall.sh
```

حذف کامل تنظیمات، state و logها:

```bash
sudo bash scripts/uninstall.sh --purge
```

## امنیت

- برای هر نود token یکتا و قابل لغو بسازید؛
- فایل `/etc/xnode/agent.json` را با mode `0600` نگه دارید؛
- فقط پورت inboundهای Xray را در firewall باز کنید؛
- پورت‌های `10085` و `19090` را عمومی نکنید؛
- token یا UUID واقعی را داخل issue، screenshot یا repository نگذارید؛
- ساعت سرور را با NTP درست نگه دارید؛
- قبل از upgrade مهم از `/etc/xnode` و `/var/lib/xnode` نسخهٔ پشتیبان بگیرید.

## عیب‌یابی سریع

اگر سرویس بالا نمی‌آید:

```bash
sudo systemctl status xnode-agent --no-pager
sudo journalctl -u xnode-agent -n 200 --no-pager
sudo /usr/local/bin/xnode-agent -config /etc/xnode/agent.json
```

اگر پنل ایجنت را نمی‌بیند، `panel_url`، token، DNS، firewall و پاسخ endpointها را بررسی کنید. اگر مصرف ارسال نمی‌شود، محتویات `traffic-spool` و قابلیت deduplicate پنل را بررسی کنید. اگر `strict_limits_ready` برابر false است، مطمئن شوید هستهٔ release نصب شده و `require_patched_core` غیرفعال نشده است.

## اسناد تخصصی فارسی

- [نصب و نگهداری](docs/fa/DEPLOYMENT.md)
- [قرارداد اتصال پنل](docs/fa/PANEL_API.md)
- [محدودیت‌ها و sessionها](docs/fa/LIMITS.md)
- [تحویل امن آمار ترافیک](docs/fa/TRAFFIC_DELIVERY.md)
- [معماری](docs/fa/ARCHITECTURE.md)
- [رفتار API زمان اجرای Xray](docs/fa/RUNTIME_API.md)

</div>

