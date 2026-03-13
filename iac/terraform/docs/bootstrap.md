# Bootstrap

One-time setup for GCS state backend.

## What it creates

- GCP project `gh-jae-labs-terraform`
- GCS bucket `gh-jae-labs-terraform` with versioning
- Service account `terraform` with `storage.objectAdmin` on the bucket
- Key file `gcp-sa-key.json` (gitignored)

## Run

```bash
# requires gcloud CLI authenticated
bash scripts/bootstrap.sh
```

## Environment variables

Override defaults via:

| Variable | Default | Purpose |
|---|---|---|
| `GCP_PROJECT_ID` | `gh-jae-labs-terraform` | GCP project ID |
| `TF_STATE_BUCKET` | `gh-jae-labs-terraform` | GCS bucket name |
| `GCP_REGION` | `us-central1` | Bucket region |

After bootstrap, add to your shell:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=./gcp-sa-key.json
```
