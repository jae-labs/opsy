# CI/CD

GitHub Actions workflows at `.github/workflows/` in the repo root (`jae-labs/opsy`). Each root module has a dedicated apply workflow (e.g. `github-apply.yml`, `cloudflare-apply.yml`, `doppler-apply.yml`) that calls the shared `terraform-reusable.yml` workflow.

## Trigger

Push to `main` affecting `iac/terraform/github/**` or `iac/terraform/modules/github/**`.

## Secrets

Stored in the `production` environment on the `opsy` repo.

| Secret | Value |
|---|---|
| `GH_PAT` | Fine-grained PAT with org admin permissions |
| `GCP_SA_KEY` | Raw JSON contents of `gcp-sa-key.json` |

## Flow

1. Checkout code
2. Write GCP credentials to temp file
3. `terraform init`
4. `terraform apply -auto-approve`
5. Cleanup credentials

## Security

- Secrets scoped to `production` environment only
- Environment restricted to protected branches (`main`)
- Branch protection requires PR with 1 approval before merge
- No direct pushes to `main`
