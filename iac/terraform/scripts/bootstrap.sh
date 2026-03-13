#!/usr/bin/env bash
# creates GCP project, GCS state bucket, and Terraform service account.
# run once before terraform init.
set -euo pipefail

PROJECT_ID="${GCP_PROJECT_ID:-gh-jae-labs-terraform}"
BUCKET_NAME="${TF_STATE_BUCKET:-gh-jae-labs-terraform}"
SA_NAME="terraform"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
KEY_FILE="./gcp-sa-key.json"
REGION="${GCP_REGION:-us-central1}"

echo "==> creating GCP project ${PROJECT_ID}"
if gcloud projects describe "${PROJECT_ID}" &>/dev/null; then
  echo "    project already exists, continuing"
else
  gcloud projects create "${PROJECT_ID}" --name="jae-labs Terraform"
fi

echo "==> setting active project"
gcloud config set project "${PROJECT_ID}"

echo "==> linking billing account"
BILLING_ACCOUNT=$(gcloud billing accounts list --format="value(ACCOUNT_ID)" --filter="open=true" | head -n 1)
if [ -z "${BILLING_ACCOUNT}" ]; then
  echo "    ERROR: no active billing account found. Set one up at https://console.cloud.google.com/billing"
  exit 1
fi
CURRENT_BILLING=$(gcloud billing projects describe "${PROJECT_ID}" --format="value(billingAccountName)" 2>/dev/null || true)
if [ -n "${CURRENT_BILLING}" ]; then
  echo "    billing already linked: ${CURRENT_BILLING}"
else
  gcloud billing projects link "${PROJECT_ID}" --billing-account="${BILLING_ACCOUNT}"
  echo "    linked to ${BILLING_ACCOUNT}"
fi

echo "==> enabling APIs"
gcloud services enable storage.googleapis.com iam.googleapis.com

echo "==> creating GCS bucket ${BUCKET_NAME}"
if gcloud storage buckets describe "gs://${BUCKET_NAME}" &>/dev/null; then
  echo "    bucket already exists, continuing"
else
  gcloud storage buckets create "gs://${BUCKET_NAME}" \
    --project="${PROJECT_ID}" \
    --location="${REGION}" \
    --uniform-bucket-level-access
fi

echo "==> enabling object versioning"
gcloud storage buckets update "gs://${BUCKET_NAME}" --versioning

echo "==> creating service account ${SA_NAME}"
if gcloud iam service-accounts describe "${SA_EMAIL}" &>/dev/null; then
  echo "    service account already exists, continuing"
else
  gcloud iam service-accounts create "${SA_NAME}" \
    --display-name="Terraform"
fi

echo "==> granting objectAdmin on bucket"
gcloud storage buckets add-iam-policy-binding "gs://${BUCKET_NAME}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/storage.objectAdmin"

if [ -f "${KEY_FILE}" ] && [ -s "${KEY_FILE}" ]; then
  echo "    key file already exists at ${KEY_FILE}, skipping"
else
  rm -f "${KEY_FILE}"
  echo "==> creating key file ${KEY_FILE}"
  gcloud iam service-accounts keys create "${KEY_FILE}" \
    --iam-account="${SA_EMAIL}"
fi

echo ""
echo "Bootstrap complete."
echo "Key written to: ${KEY_FILE}"
echo "Add to .env: export GOOGLE_APPLICATION_CREDENTIALS=${KEY_FILE}"
