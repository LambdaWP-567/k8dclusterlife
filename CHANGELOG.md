# 1.0.0 (2026-06-14)


### Bug Fixes

* skip RequireAuth when no OIDC providers are configured ([3baf1af](https://github.com/LambdaWP-567/k8dclusterlife/commit/3baf1af6c08732d5b33a8d1eba3aa66083b13be4))
* upgrade to Go 1.25, fix docker-build-push-action parameter ([f89d23d](https://github.com/LambdaWP-567/k8dclusterlife/commit/f89d23d81f6a54a4c016e89aa1319d2f7d6f7633))


### Features

* add cluster management (upload kubeconfig, K8s Secret storage) ([338bd0f](https://github.com/LambdaWP-567/k8dclusterlife/commit/338bd0fcec19989af579cc975bcfaacb161747d7))
* add Helm chart with full Kubernetes deployment templates ([71e50b7](https://github.com/LambdaWP-567/k8dclusterlife/commit/71e50b7703a2600f8bd3a085ff909ee9e0f715c9))
* add KI-Healing Agent (Claude Tool Use agentic loop) ([2bdb869](https://github.com/LambdaWP-567/k8dclusterlife/commit/2bdb869549893bcc70deb9309d64ca3206850d8d))
* add multi-provider OIDC auth (Entra ID, GitHub, Google) ([76ab9b8](https://github.com/LambdaWP-567/k8dclusterlife/commit/76ab9b828cc2884d87549bd6b02dd364b31ae8b2))
* add Teams webhook notifications and Prometheus /metrics endpoint ([5c58534](https://github.com/LambdaWP-567/k8dclusterlife/commit/5c58534bda5757f6b5b36f31a2a39067638c5ac2))
* add test ecosystem (scenarios, controller, Playwright E2E tests) ([a2858f5](https://github.com/LambdaWP-567/k8dclusterlife/commit/a2858f58b012c9330cfb69c738508291c75f3170))
* add WebSocket live updates, Settings page, and /api/settings endpoint ([d6348f2](https://github.com/LambdaWP-567/k8dclusterlife/commit/d6348f29fe8580b065139ed194c8e89e91d6ea47))
* DB schema, Redis cache, cluster controller + status mapper ([339e5e3](https://github.com/LambdaWP-567/k8dclusterlife/commit/339e5e35dbc24650598cdf59a5dd1f3b7121cf49))
* frontend MVP dashboard — problem list, status badges, dark mode ([216deaa](https://github.com/LambdaWP-567/k8dclusterlife/commit/216deaab57efa56ff398f393775f7c0eadc0c2de))
* project scaffolding — Go module, React apps, Docker, CI/CD ([2f33275](https://github.com/LambdaWP-567/k8dclusterlife/commit/2f33275460f343ebf4b03a0bc97ea8366694fb09))
* wire PostgreSQL store into server (cluster management, settings, migrations) ([7bc9056](https://github.com/LambdaWP-567/k8dclusterlife/commit/7bc9056f61cabce478fb7018448d2bd9737d2e29))
