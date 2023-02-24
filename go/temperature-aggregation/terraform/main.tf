/**
 * Copyright 2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "firestore_location" {
  type    = string
  default = "us-central"
}

terraform {
  required_version = "~> 1.3.7"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.53.1"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

data "google_project" "project" {
  project_id = var.project_id
}

/*
 * Google Cloud APIs
 *
 * Enable APIs that Temperature Aggregation uses.
 */

resource "google_project_service" "artifactregistry" {
  service = "artifactregistry.googleapis.com"
}

resource "google_project_service" "bigquery" {
  service = "bigquery.googleapis.com"
}

resource "google_project_service" "cloudbuild" {
  service = "cloudbuild.googleapis.com"
}

resource "google_project_service" "cloudscheduler" {
  service = "cloudscheduler.googleapis.com"
}

resource "google_project_service" "iam" {
  service = "iam.googleapis.com"
}

resource "google_project_service" "pubsub" {
  service = "pubsub.googleapis.com"
}

resource "google_project_service" "run" {
  service = "run.googleapis.com"
}

resource "google_project_service" "sourcerepo" {
  service = "sourcerepo.googleapis.com"
}

/*
 * Firestore
 *
 * Aggregator needs Firestore with Datastore mode.
 */

resource "google_app_engine_application" "app" {
  project       = data.google_project.project.project_id
  location_id   = var.firestore_location
  database_type = "CLOUD_DATASTORE_COMPATIBILITY"
}

resource "google_project_iam_member" "aggregator-user" {
  project = google_app_engine_application.app.project
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.aggregator.email}"
}

/*
 * BigQuery
 *
 * Create a dataset and a table to save measured temperature records.
 */

resource "google_bigquery_dataset" "raw_data" {
  dataset_id  = "raw_data"
  description = "raw_data dataset contains tables which store measured raw data"
  location    = var.region

  depends_on = [google_project_service.bigquery]
}

resource "google_bigquery_table" "temperature" {
  dataset_id  = google_bigquery_dataset.raw_data.dataset_id
  table_id    = "temperature"
  description = "records of measured temperature"

  time_partitioning {
    field                    = "publish_time"
    type                     = "HOUR"
    require_partition_filter = true
  }

  schema = file("./table.schema.json")
}

resource "google_bigquery_table_iam_member" "aggregator-dataViewer" {
  dataset_id = google_bigquery_table.temperature.dataset_id
  table_id   = google_bigquery_table.temperature.table_id
  role       = "roles/bigquery.dataViewer"
  member     = "serviceAccount:${google_service_account.aggregator.email}"
}

resource "google_bigquery_table_iam_member" "pubsub-dataEditor" {
  dataset_id = google_bigquery_table.temperature.dataset_id
  table_id   = google_bigquery_table.temperature.table_id
  role       = "roles/bigquery.dataEditor"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"

  depends_on = [google_project_service.pubsub]
}

resource "google_bigquery_table_iam_member" "pubsub-metadataViewer" {
  dataset_id = google_bigquery_table.temperature.dataset_id
  table_id   = google_bigquery_table.temperature.table_id
  role       = "roles/bigquery.metadataViewer"
  member     = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"

  depends_on = [google_project_service.pubsub]
}

resource "google_project_iam_member" "aggregator-jobUser" {
  project = google_bigquery_table.temperature.project
  role    = "roles/bigquery.jobUser"
  member  = "serviceAccount:${google_service_account.aggregator.email}"
}

/*
 * Pub/Sub
 *
 * Create a topic and a BigQuery subscription to insert messages into BigQuery table.
 */

resource "google_pubsub_schema" "temperature" {
  name       = "temperature"
  type       = "AVRO"
  definition = file("./topic.schema.json")

  depends_on = [google_project_service.pubsub]
}

resource "google_pubsub_topic" "temperature" {
  name = "temperature"
  schema_settings {
    schema   = google_pubsub_schema.temperature.id
    encoding = "JSON"
  }
}

resource "google_pubsub_topic_iam_member" "temperature-publisher" {
  topic  = google_pubsub_topic.temperature.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${google_service_account.receiver.email}"
}

resource "google_pubsub_subscription" "temperature-bigquery" {
  name  = "temperature-bigquery"
  topic = google_pubsub_topic.temperature.name

  bigquery_config {
    table            = "${google_bigquery_table.temperature.project}:${google_bigquery_table.temperature.dataset_id}.${google_bigquery_table.temperature.table_id}"
    use_topic_schema = true
    write_metadata   = true
  }

  depends_on = [
    google_bigquery_table_iam_member.pubsub-dataEditor,
    google_bigquery_table_iam_member.pubsub-metadataViewer
  ]
}

/*
 * Receiver (Cloud Run app)
 *
 * Receiver receives a measured temperature series as a HTTP POST request
 * from an edge thermometer, unpacks it into flat data points and publishes
 * them to a Pub/Sub topic.
 */

resource "google_service_account" "receiver" {
  account_id   = "receiver"
  display_name = "Service Account for Receiver app"
}

resource "google_service_account_iam_member" "receiver-user" {
  service_account_id = google_service_account.receiver.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.receiver-build.email}"
}

resource "google_cloud_run_service" "receiver" {
  name     = "receiver"
  location = var.region

  template {
    spec {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/hello" // dummy image
        env {
          name  = "PUBSUB_PROJECT_ID"
          value = google_pubsub_topic.temperature.project
        }
        env {
          name  = "PUBSUB_TOPIC_ID"
          value = google_pubsub_topic.temperature.name
        }
      }
      container_concurrency = 10
      timeout_seconds       = 60
      service_account_name  = google_service_account.receiver.email
    }
  }

  depends_on = [google_project_service.run]

  lifecycle {
    ignore_changes = [
      template[0].spec[0].containers[0].image
    ]
  }
}

resource "google_cloud_run_service_iam_member" "receiver-invoker" {
  service  = google_cloud_run_service.receiver.name
  location = google_cloud_run_service.receiver.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "receiver-developer" {
  service  = google_cloud_run_service.receiver.name
  location = google_cloud_run_service.receiver.location
  role     = "roles/run.developer"
  member   = "serviceAccount:${google_service_account.receiver-build.email}"
}

/*
 * Aggregator (Cloud Run jobs app)
 *
 * Aggregator fetches hourly temperature data from BigQuery and uploads it
 * to Cloud Storage. It stores the last execution time in Cloud Firestore.
 * It's triggered by Cloud Scheduler.
 */

resource "google_service_account" "aggregator" {
  account_id   = "aggregator"
  display_name = "Service Account for Aggregator app"
}

resource "google_service_account_iam_member" "aggregator-user" {
  service_account_id = google_service_account.aggregator.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.aggregator-build.email}"
}

resource "google_service_account" "aggregator-scheduler" {
  account_id   = "aggregator-scheduler"
  display_name = "Service Account for Aggregator scheduler"
}

resource "google_cloud_run_v2_job" "aggregator" {
  name         = "aggregator"
  location     = var.region
  launch_stage = "BETA"

  template {
    parallelism = 1
    task_count  = 1

    template {
      service_account = google_service_account.aggregator.email
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
        env {
          name  = "BIGQUERY_PROJECT_ID"
          value = google_bigquery_table.temperature.project
        }
        env {
          name  = "BIGQUERY_DATASET_ID"
          value = google_bigquery_table.temperature.dataset_id
        }
        env {
          name  = "BIGQUERY_TABLE_ID"
          value = google_bigquery_table.temperature.table_id
        }
        env {
          name  = "STORAGE_BUCKET_NAME"
          value = google_storage_bucket.temperature.name
        }
        env {
          name  = "DATASTORE_PROJECT_ID"
          value = google_app_engine_application.app.project
        }
      }
      max_retries = 3
    }
  }

  depends_on = [google_project_service.run]

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image
    ]
  }
}

resource "google_cloud_run_v2_job_iam_member" "aggregator-invoker" {
  name     = google_cloud_run_v2_job.aggregator.name
  location = google_cloud_run_v2_job.aggregator.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.aggregator-scheduler.email}"
}

resource "google_cloud_run_v2_job_iam_member" "aggregator-developer" {
  name     = google_cloud_run_v2_job.aggregator.name
  location = google_cloud_run_v2_job.aggregator.location
  role     = "roles/run.developer"
  member   = "serviceAccount:${google_service_account.aggregator-build.email}"
}

resource "google_cloud_scheduler_job" "aggregator-scheduler" {
  name        = "aggregator"
  description = "Hourly trigger for Aggregator job"
  schedule    = "10 * * * *"
  time_zone   = "UTC"

  http_target {
    http_method = "POST"
    uri         = "https://${google_cloud_run_v2_job.aggregator.location}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${google_cloud_run_v2_job.aggregator.project}/jobs/${google_cloud_run_v2_job.aggregator.name}:run"

    oauth_token {
      service_account_email = google_service_account.aggregator-scheduler.email
    }
  }

  depends_on = [google_project_service.cloudscheduler]
}

resource "google_storage_bucket" "temperature" {
  name          = "${data.google_project.project.project_id}-temperature"
  location      = var.region
  storage_class = "STANDARD"
  autoclass {
    enabled = true
  }
  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "aggregator-objectCreator" {
  bucket = google_storage_bucket.temperature.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.aggregator.email}"
}

/*
 * Source Repository
 *
 * Receiver, Aggregator and this Terraform code are stored in a repository
 * as a monorepo.
 */

resource "google_sourcerepo_repository" "temperature-aggregation" {
  name = "temperature-aggregation"

  depends_on = [google_project_service.sourcerepo]
}

resource "google_sourcerepo_repository_iam_member" "receiver-build-reader" {
  repository = google_sourcerepo_repository.temperature-aggregation.name
  role       = "roles/source.reader"
  member     = "serviceAccount:${google_service_account.receiver-build.email}"
}

resource "google_sourcerepo_repository_iam_member" "aggregator-build-reader" {
  repository = google_sourcerepo_repository.temperature-aggregation.name
  role       = "roles/source.reader"
  member     = "serviceAccount:${google_service_account.aggregator-build.email}"
}

/*
 * Artifact Registry
 *
 * Docker images for Receiver and Aggregator are stored in a repository of
 * Artifact Registry.
 */

resource "google_artifact_registry_repository" "receiver" {
  location      = var.region
  repository_id = "receiver"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}

resource "google_artifact_registry_repository_iam_member" "receiver-build-writer" {
  location   = google_artifact_registry_repository.receiver.location
  repository = google_artifact_registry_repository.receiver.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.receiver-build.email}"
}

resource "google_artifact_registry_repository_iam_member" "receiver-reader" {
  location   = google_artifact_registry_repository.receiver.location
  repository = google_artifact_registry_repository.receiver.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.receiver.email}"
}

resource "google_artifact_registry_repository" "aggregator" {
  location      = var.region
  repository_id = "aggregator"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}

resource "google_artifact_registry_repository_iam_member" "aggregator-build-writer" {
  location   = google_artifact_registry_repository.aggregator.location
  repository = google_artifact_registry_repository.aggregator.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.aggregator-build.email}"
}

resource "google_artifact_registry_repository_iam_member" "aggregator-writer" {
  location   = google_artifact_registry_repository.aggregator.location
  repository = google_artifact_registry_repository.aggregator.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.aggregator.email}"
}

/*
 * CI/CD
 *
 * Configure Cloud Build and Artifact Registry for Receiver and Aggregator.
 */

resource "google_service_account" "receiver-build" {
  account_id   = "receiver-build"
  display_name = "Cloud Build Service Account for deploying Receiver"
}

resource "google_project_iam_member" "receiver-build-logWriter" {
  project = data.google_project.project.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.receiver-build.email}"
}

resource "google_cloudbuild_trigger" "receiver-build" {
  name            = "receiver"
  filename        = "receiver/cloudbuild.yaml"
  service_account = google_service_account.receiver-build.id
  included_files  = ["receiver/**"]

  trigger_template {
    repo_name   = google_sourcerepo_repository.temperature-aggregation.name
    branch_name = "^main$"
    dir         = "receiver/"
  }

  substitutions = {
    _REPOSITORY  = "${google_artifact_registry_repository.receiver.location}-docker.pkg.dev/${google_artifact_registry_repository.receiver.project}/${google_artifact_registry_repository.receiver.repository_id}"
    _RUN_REGION  = google_cloud_run_service.receiver.location
    _RUN_SERVICE = google_cloud_run_service.receiver.name
  }

  depends_on = [google_project_service.cloudbuild]
}

resource "google_service_account" "aggregator-build" {
  account_id   = "aggregator-build"
  display_name = "Cloud Build Service Account for deploying Aggregator"
}


resource "google_project_iam_member" "aggregator-build-logWriter" {
  project = data.google_project.project.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.aggregator-build.email}"
}

resource "google_cloudbuild_trigger" "aggregator-build" {
  name            = "aggregator"
  filename        = "aggregator/cloudbuild.yaml"
  service_account = google_service_account.aggregator-build.id
  included_files  = ["aggregator/**"]

  trigger_template {
    repo_name   = google_sourcerepo_repository.temperature-aggregation.name
    branch_name = "^main$"
    dir         = "aggregator/"
  }

  substitutions = {
    _REPOSITORY = "${google_artifact_registry_repository.aggregator.location}-docker.pkg.dev/${google_artifact_registry_repository.aggregator.project}/${google_artifact_registry_repository.aggregator.repository_id}"
    _RUN_REGION = google_cloud_run_v2_job.aggregator.location
    _RUN_JOB    = google_cloud_run_v2_job.aggregator.name
  }

  depends_on = [google_project_service.cloudbuild]
}
