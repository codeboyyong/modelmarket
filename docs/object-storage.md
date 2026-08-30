# Object Storage

Local development remains filesystem-backed:

```text
OBJECT_STORAGE_PROVIDER=local
OBJECT_STORAGE_DIR=tmp/storage
MM_ASSET_BUCKET=model-market-dev-assets
MM_ASSET_PUBLIC_URL=http://localhost:8080/api/v1/mock-s3
```

Real S3 mode uses the AWS SDK credential chain and presigned PUT/GET URLs:

```text
OBJECT_STORAGE_PROVIDER=s3
MM_ASSET_BUCKET=your-private-bucket
MM_ASSET_PUBLIC_URL=
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_SESSION_TOKEN=
AWS_REGION=us-east-1
S3_ENDPOINT_URL=
S3_FORCE_PATH_STYLE=false
S3_PRESIGN_MINUTES=15
```

Leave `MM_ASSET_PUBLIC_URL` empty for a private bucket. The API then refreshes
download URLs when assets are listed. `S3_ENDPOINT_URL` and path-style mode are
optional and support S3-compatible development services. The IAM identity
needs `s3:GetObject`, `s3:PutObject`, and `s3:DeleteObject` on the configured
bucket prefix. Generated media and browser uploads use the selected provider;
text chat is persisted as conversation messages in PostgreSQL.

Configure bucket CORS to allow the frontend origin to send `PUT` requests with
the generated `Content-Type` header. Keep public access blocked when using
presigned GET URLs.
