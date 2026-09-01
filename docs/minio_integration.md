# MinIO integration

The application uses MinIO through its S3-compatible API. The MinIO console
and S3 API are different services:

- Console: `http://100.85.190.123:9001` (administration only)
- S3 API: `http://100.85.190.123:9000` (application endpoint)

Do not put port `9001` or `/login` in `S3_ENDPOINT_URL`.

## 1. Create or confirm the bucket

In the MinIO console, create a private bucket named
`model-market-dev-assets`, or use the name of an existing private bucket in
`MM_ASSET_BUCKET`.

The Access Key must be allowed to perform these operations on the bucket:

- `s3:GetObject`
- `s3:PutObject`
- `s3:DeleteObject`

## 2. Configure `.env`

Use one object-storage section in the repository-root `.env`:

```dotenv
OBJECT_STORAGE_PROVIDER=s3
MM_ASSET_BUCKET=model-market-dev-assets
MM_ASSET_PUBLIC_URL=

AWS_ACCESS_KEY_ID=REPLACE_WITH_MINIO_ACCESS_KEY
AWS_SECRET_ACCESS_KEY=REPLACE_WITH_MINIO_SECRET_KEY
AWS_SESSION_TOKEN=
AWS_REGION=us-east-1

S3_ENDPOINT_URL=http://100.85.190.123:9000
S3_FORCE_PATH_STYLE=true
S3_PRESIGN_MINUTES=15
```

`AWS_REGION` is required by the AWS SDK; `us-east-1` is the usual MinIO
default. A permanent MinIO Access Key and Secret Key do not use an AWS session
token, so `AWS_SESSION_TOKEN` stays empty. Path-style addressing should be
enabled for MinIO.

Remove duplicate definitions elsewhere in `.env`. In particular, do not leave
this local-development value active:

```dotenv
MM_ASSET_PUBLIC_URL=http://localhost:8080/api/v1/mock-s3
```

Leaving `MM_ASSET_PUBLIC_URL` empty makes the backend return temporary,
presigned MinIO download URLs. Never commit `.env` or the MinIO credentials.

## 3. Configure bucket CORS

Browser uploads go directly to the presigned MinIO URL. Configure the bucket's
CORS rules to allow every frontend origin that will use the application. For
local development that is normally `http://localhost:3000`.

Allow at least:

- Methods: `GET`, `PUT`, `HEAD`
- Request headers: `Content-Type` (using `*` for allowed headers is also fine)
- Exposed headers: `ETag`

Do not make the bucket public; presigned URLs provide temporary access.

## 4. Network requirements

Both the backend and the user's browser must be able to reach
`http://100.85.190.123:9000`. This matters because upload and download URLs are
signed with that host and returned to the browser. If this is a Tailscale IP,
each client using the web application must have access to that Tailscale
network.

When the frontend is served over HTTPS, browsers will block uploads to an HTTP
MinIO endpoint as mixed content. Production deployments should expose MinIO
through HTTPS and set `S3_ENDPOINT_URL` to that HTTPS API address.

## 5. Restart and verify

Restart the backend after editing `.env`:

```sh
docker compose up -d --build backend
docker compose logs -f backend
```

For a backend started directly, stop and restart the Go server instead. The
AWS SDK reads the Access Key and Secret Key automatically. At startup the code
constructs the S3 client, but it does not contact MinIO or confirm that the
bucket exists until the first object operation.

Verify the integration by uploading a small image through the application,
then check all of the following:

1. The browser's upload request targets port `9000`, not `9001`.
2. The upload request returns a successful `2xx` response.
3. The object appears in the configured MinIO bucket.
4. Opening the asset uses a temporary MinIO URL and downloads successfully.
5. Deleting the asset removes the object from the bucket.

Common failures:

| Symptom | Likely cause |
| --- | --- |
| Connection refused | Port `9000` is not exposed or reachable |
| `AccessDenied` | Access Key policy lacks bucket/object permissions |
| `NoSuchBucket` | `MM_ASSET_BUCKET` does not exist |
| Browser CORS error | Bucket CORS does not allow the frontend origin or `PUT` |
| Signature mismatch | Wrong Secret Key, endpoint, system time, or modified signed headers |
| URL points to `/api/v1/mock-s3` | A duplicate/non-empty `MM_ASSET_PUBLIC_URL` is still active |
| URL contains a Docker-only hostname | The browser cannot resolve the endpoint used for signing |
