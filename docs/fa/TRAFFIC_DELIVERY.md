<div dir="rtl">

# تحویل پایدار ترافیک و محاسبهٔ مصرف

[English](../TRAFFIC_DELIVERY.md) | [فارسی](TRAFFIC_DELIVERY.md)

شمارنده‌های کاربران Xray با گزینهٔ reset خوانده می‌شوند، بنابراین هر گزارش فقط delta جدید است. اگر بعد از reset ارتباط شبکه قطع شود، نباید مصرف از بین برود.

## روند ارسال

1. ایجنت آمار Xray را همراه reset می‌خواند.
2. یک `TrafficBatch` با `event_id` تصادفی می‌سازد.
3. batch ابتدا با temp-file و rename اتمیک در `traffic_spool_dir` ذخیره می‌شود.
4. batch به پنل POST می‌شود.
5. فقط پس از پاسخ موفق HTTP، فایل محلی حذف می‌شود.
6. batchهای باقی‌مانده در چرخهٔ بعد، پیش از دریافت desired state، دوباره ارسال می‌شوند.

```json
{
  "event_id":"d0a48cda15c14a68a6251d85a0c7af91",
  "node_id":"node-12",
  "collected_at":1786680000,
  "records":[
    {
      "name":"user>>>u:25|i:101>>>traffic>>>downlink",
      "value":2040,
      "user_id":"25",
      "inbound_id":"101",
      "direction":"downlink"
    }
  ]
}
```

## الزام پنل: Idempotency

پنل باید claim یا ذخیرهٔ `event_id` و اضافه‌کردن deltaها را در یک transaction انجام دهد. اگر همان ID قبلاً پردازش شده است، بدون افزودن دوبارهٔ مصرف پاسخ موفق بدهد.

این کار حالتی را پوشش می‌دهد که پنل batch را commit کرده ولی پاسخ HTTP در شبکه گم شده و ایجنت دوباره ارسال می‌کند.

## فاصلهٔ بسیار کوچک crash

بین reset اتمیک شمارندهٔ Xray و ذخیرهٔ batch روی دیسک یک فاصلهٔ بسیار کوچک اجتناب‌ناپذیر وجود دارد. حذف کامل آن به API حسابداری مبتنی بر acknowledgement داخل هسته نیاز دارد. spool فعلی حالت رایج‌ترِ قطع شبکه و خطای تحویل را پوشش می‌دهد.

</div>

