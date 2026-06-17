# Local E2E Demo

This demo runs entirely on the developer machine with PostgreSQL, the Go backend, the TypeScript frontend, mock payment, mock media generation, and local mock S3 storage.

## Start

```sh
scripts/dev-up.sh
scripts/init_db.sh dev
scripts/populate_test_data.sh dev
scripts/demo-smoke.sh dev
```

Open:

- Frontend: http://localhost:3000
- Backend: http://localhost:8080/healthz

## What Is Mocked

- Payment: `/api/v1/credits/purchase` posts credits immediately when `payment_provider_mode=mock` and `payment_mock_enabled=true`.
- Image generation: media requests create SVG files under `OBJECT_STORAGE_DIR`.
- Video/audio generation: media requests create JSON mock manifests under `OBJECT_STORAGE_DIR`.
- S3: database rows keep S3-shaped fields (`bucket_name`, `object_key`, `storage_path`), while downloads use `MM_ASSET_PUBLIC_URL=http://localhost:8080/api/v1/mock-s3`.

## Demo Path

1. Log in as `yong_zhao@example.com` with the seeded password from `docs/demo-accounts.md`.
2. Use Buy Credit to add mock credits.
3. Go to Workbench.
4. Pick an image, video, or audio model.
5. Open output parameters and choose size/resolution/length options.
6. Send a prompt.
7. Confirm the Artifacts section shows a generated item and a mock S3 download URL.
8. Open Credit usage or Pricing to confirm balance and usage changed.
9. Log in as `admin@example.com` and open Admin for system route/config/balance overview.

## Smoke Script

`scripts/demo-smoke.sh dev` validates the backend path without the browser:

- health and readiness
- mock payment purchase
- API key creation
- mock image generation
- artifact download through mock S3
- database row lookup
