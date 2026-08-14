<div dir="rtl">

# قرارداد API پنل — نسخه ۱.۰

[English](../PANEL_API.md) | [فارسی](PANEL_API.md)

پنل منبع حقیقت desired configuration و مصرف تجمیعی commit‌شده است. هر درخواست ایجنت این header را دارد:

```http
Authorization: Bearer <unique node token>
```

پنل باید token را به همان `node_id` محدود کند و برای هر نود token متفاوت داشته باشد.

## دریافت وضعیت مطلوب

```http
GET /api/v1/nodes/{node_id}/desired-state
```

نمونهٔ کامل سطح بالا:

```json
{
  "version":"100",
  "enabled":true,
  "mode":"active",
  "node":{
    "region":"DE",
    "group":"premium",
    "tags":["reality","10g"],
    "weight":100,
    "traffic_threshold_bytes":21990232555520,
    "traffic_used_bytes":12000000000000,
    "drain_target_online":0
  },
  "inbounds":[],
  "outbounds":[],
  "routing":{},
  "dns":{}
}
```

### حالت‌های نود

- `active`: همگام‌سازی و سرویس‌دهی عادی؛
- `draining`: حفظ config، رد اتصال جدید، اجازهٔ پایان اتصال قبلی و گزارش `drain_ready`؛
- `maintenance`: جمع‌آوری/گزارش وضعیت ولی توقف Xray؛
- `disabled`: توقف Xray و باقی‌ماندن در حالت غیرفعال.

اگر `traffic_used_bytes >= traffic_threshold_bytes` شود، ایجنت حتی با mode درخواستی active به‌صورت محلی وارد draining می‌شود.

### Inbound

```json
{
  "id":"101",
  "tag":"vless-reality-443",
  "listen":"0.0.0.0",
  "port":443,
  "protocol":"vless",
  "ip_limit_mode":"source",
  "settings":{"decryption":"none"},
  "stream_settings":{},
  "sniffing":{},
  "users":[]
}
```

JSON وابسته به protocol مستقیماً به Xray منتقل می‌شود. تزریق کاربران برای VLESS، VMess، Trojan و Shadowsocks خودکار است و userهای WireGuard به peer تبدیل می‌شوند. protocolهای دیگر را می‌توان با settings بومی Xray ساخت.

### Credential کاربر یا دستگاه

```json
{
  "id":"device-25-a",
  "account_id":"25",
  "enabled":true,
  "session_generation":3,
  "outbound_tag":"direct",
  "level":0,
  "credential":{
    "id":"550e8400-e29b-41d4-a716-446655440000",
    "flow":"xtls-rprx-vision"
  },
  "limits":{
    "traffic_bytes":536870912000,
    "traffic_used_bytes":1234567890,
    "upload_bps":2500000,
    "download_bps":12500000,
    "ip_limit":2,
    "device_limit":3,
    "connection_limit":20,
    "expires_at":0
  }
}
```

`outbound_tag` یک قانون routing خودکار برای هویت این کاربر می‌سازد. برای قطع sessionهای فعلی بدون غیرفعال‌کردن credential، `session_generation` را افزایش دهید.

## ارسال Heartbeat

```http
POST /api/v1/nodes/{node_id}/heartbeat
```

```json
{
  "node_id":"node-12",
  "agent_version":"1.0.0",
  "healthy":true,
  "xray_running":true,
  "xray_api_healthy":true,
  "cpu_percent":13.2,
  "memory_bytes":734003200,
  "load1":0.21,
  "network_rx_bytes":123,
  "network_tx_bytes":456,
  "network_rx_bps":800000000,
  "network_tx_bps":120000000,
  "online_users":14,
  "tracked_ips":19,
  "mode":"draining",
  "drain_ready":false,
  "region":"DE",
  "group":"premium",
  "tags":["reality","10g"],
  "weight":100,
  "strict_limits_ready":true,
  "state_version":"100",
  "message":"ok"
}
```

مقادیر `network_*_bps` در heartbeat بر حسب **بیت‌برثانیه** هستند. `strict_limits_ready` وجود marker هستهٔ patch‌شده را هنگام نیاز به محدودیت سخت‌گیرانه تأیید می‌کند.

Scheduler سراسری پنل می‌تواند health، mode، drain، weight، region/group/tags و policy ظرفیت خودش را برای failover و load balancing استفاده کند.

## ارسال ترافیک

```http
POST /api/v1/nodes/{node_id}/traffic
```

```json
{
  "event_id":"d0a48cda15c14a68a6251d85a0c7af91",
  "node_id":"node-12",
  "collected_at":1786680000,
  "records":[
    {
      "name":"user>>>u:device-25-a|i:101>>>traffic>>>downlink",
      "value":2040,
      "user_id":"device-25-a",
      "inbound_id":"101",
      "direction":"downlink"
    }
  ]
}
```

پنل **باید** `event_id` را transactionally deduplicate و سپس هر `value` را به‌عنوان delta اضافه کند. یک event ممکن است پس از خطای مبهم شبکه retry شود.

## ارسال Sessionها

وقتی `report_sessions:true` است:

```http
POST /api/v1/nodes/{node_id}/sessions
```

```json
{
  "node_id":"node-12",
  "generated_at":1786680000,
  "window_sec":120,
  "records":[
    {
      "user_id":"device-25-a",
      "inbound_id":"101",
      "ips":["203.0.113.10"],
      "last_seen":1786679996,
      "recent_connections":6,
      "source":"xray-online"
    }
  ],
  "violations":[
    {
      "user_id":"device-25-a",
      "inbound_id":"101",
      "reason":"ip_limit",
      "observed":3,
      "limit":2
    }
  ]
}
```

دلیل‌های policy شامل `expired`، `traffic_quota`، `ip_limit` و `device_limit` هستند. IPهای `xray-online` فعال‌اند؛ `recent_connections` فقط فعالیت اخیر access log است و تعداد دقیق اتصال فعال نیست. محدودیت دقیق اتصال داخل هسته اعمال می‌شود.

## ترتیب و Retry

- desired state باید بدون اثر جانبی و قابل خواندن مکرر باشد؛
- `version` را با هر تغییر به‌صورت یکنواخت تغییر دهید؛
- heartbeat و sessions باید retry را تحمل کنند؛
- ترافیک بر اساس `event_id` پردازش شود، نه زمان رسیدن؛
- پاسخ موفق فقط پس از commit پایگاه داده ارسال شود؛
- endpointها timeout کوتاه، log امن و rate limit معقول داشته باشند.

</div>

