# Isolation Mode

Isolation mode prevents your project from making external network requests, keeping all traffic local.

## Use Cases

- **Security testing**: Test how your application behaves without external dependencies
- **Offline development**: Work without internet connection
- **Performance testing**: Eliminate network latency from tests
- **Payment testing**: Prevent accidental charges to real payment gateways
- **API isolation**: Ensure no external API calls during development

## Commands

Enable isolation mode:
```bash
madock service:enable isolation
```

Disable isolation mode:
```bash
madock service:disable isolation
```

After enabling/disabling, rebuild your containers:
```bash
madock rebuild
```

## What Gets Blocked

When isolation is enabled:

| Traffic Type | Status |
|--------------|--------|
| External HTTP/HTTPS requests | ❌ Blocked |
| External API calls | ❌ Blocked |
| Package downloads (composer, npm) | ❌ Blocked |
| Internal container communication | ✅ Allowed |
| Local database connections | ✅ Allowed |
| Local Redis/Elasticsearch | ✅ Allowed |

## Important Notes

### Before enabling isolation
1. Run `madock composer install` to ensure all dependencies are downloaded
2. Download any external resources your project needs

### Troubleshooting
If your application shows errors after enabling isolation, it likely depends on external services. Check:
- Third-party API integrations
- External image/CDN URLs
- Payment gateway connections
- Analytics services

### Temporary disable
If you need to install new packages:
```bash
madock service:disable isolation
madock rebuild
madock composer require vendor/package
madock service:enable isolation
madock rebuild
```
