<div dir="rtl">

# رفتار API زمان اجرای Xray

[English](../RUNTIME_API.md) | [فارسی](RUNTIME_API.md)

ایجنت برای سازگاری با HandlerService از subcommandهای رسمی `xray api` استفاده می‌کند.

## افزودن inbound

```bash
xray api adi --server=127.0.0.1:10085 /tmp/inbound.json
```

فایل موقت شامل یک آرایهٔ inbound است:

```json
{"inbounds":[{"tag":"vless-443","protocol":"vless","port":443}]}
```

## حذف inbound

```bash
xray api rmi --server=127.0.0.1:10085 vless-443
```

## افزودن کاربر

```bash
xray api adu --server=127.0.0.1:10085 /tmp/one-user-inbound.json
```

فایل، تنظیم واقعی inbound را دارد ولی فقط شامل کاربری است که باید اضافه شود. CLI نوع inbound را می‌سازد، user را استخراج می‌کند و `AlterInbound(AddUserOperation)` را صدا می‌زند.

## حذف کاربر

```bash
xray api rmu --server=127.0.0.1:10085 -tag=vless-443 'u:25|i:101'
```

## رفتار جایگزین

config کامل desired پیش از هر تغییر runtime اعتبارسنجی می‌شود. اگر یک فرمان runtime شکست بخورد، ایجنت config کامل را اعمال و Xray را restart می‌کند. این fallback برای تفاوت نسخه‌ها و protocolها ضروری است؛ بعضی inboundها با وجود user list در JSON، رابط runtime موردنیاز را ارائه نمی‌کنند.

</div>

