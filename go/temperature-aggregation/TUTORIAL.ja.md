# Temperature Aggregator

温度センサーのデータを Cloud Run にデプロイした REST API で収集して BigQuery
に格納し、他システムとの連携のためにそのデータを 1 時間ごとのファイルにまとめるデモです。

詳細なアーキテクチャ等は [README](https://github.com/ShawnLabo/TAP/tree/main/go/temperature-aggregation#architecture) を参照してください。

## プロジェクトの選択

ハンズオンを行う Google Cloud プロジェクトを選択して **Start** をクリックしてください。

<walkthrough-project-setup></walkthrough-project-setup>

## デモ構築の流れ

1. Terraform によるプロビジョニング
  * Terraform を実行して Google Cloud の各リソースを作成します
2. アプリケーションのデプロイ
  * Terraform によって作成された Source Repositories にコードをプッシュして、Receiver と Aggregator
    のアプリケーションを Cloud Run にデプロイします
3. テスト
  * curl コマンドで擬似的な温度センサーのデータを送信します
  * BigQuery に送信したデータが格納されていることを確認します
  * 手動で Aggregator を実行して Cloud Storage にファイルが格納されていることを確認します

## Terraform によるプロビジョニング

<walkthrough-tutorial-duration duration=8></walkthrough-tutorial-duration>

Terraform ディレクトリに移動してください。

```sh
cd terraform
```

環境変数を利用してプロジェクト ID を Terraform の変数に設定します。

```sh
export TF_VAR_project_id="<walkthrough-project-id />"
```

Terraform の初期化をします。

```sh
terraform init
```

プランを確認します。

```sh
terraform plan
```

確認後、実行します。

```sh
terraform apply -auto-approve
```

元のディレクトリに移動します。

```sh
cd ..
```

## アプリケーションのデプロイ

<walkthrough-tutorial-duration duration=5></walkthrough-tutorial-duration>

Terraform によって Source Repositories のリポジトリが作成され、各アプリケーションの CI/CD
パイプラインが Cloud Build で構築されています。
そのリポジトリにコードをプッシュすることで、アプリケーションのデプロイを実施します。

現在のディレクトリで Git の初期化をします。

```sh
git init
```

リモートリポジトリを設定します。

```sh
git remote add origin \
  https://source.developers.google.com/p/<walkthrough-project-id />/r/temperature-aggregation
```

Source Repositories 用の認証を設定します。

```sh
git config credential.https://source.developers.google.com.helper gcloud.sh
```

Git のユーザーを設定します。

```sh
git config user.name "$USER"
git config user.email "$(gcloud config get account)"
```

コードをコミットしてプッシュします。

```sh
git switch -c main
git add -A
git commit -m "Initial commit"
git push -u origin main
```

Cloud Build のビルドが成功しているか確認します。

```sh
gcloud builds list --project "<walkthrough-project-id />"
```

[Cloud Build のコンソール](https://console.cloud.google.com/cloud-build/builds?project={{project-id}})でも確認できます。

## テスト

<walkthrough-tutorial-duration duration=5></walkthrough-tutorial-duration>

### **温度センサーデータの送信**

Receiver の URL を取得します。

```sh
export RECEIVER_URL="$(gcloud run services describe receiver --project <walkthrough-project-id /> --region us-central1 --format "value(status.url)")"
```

curl コマンドで擬似的な温度センサーのデータを送信します。

```sh
./tutorial/post_data.sh
```

### **BigQuery でデータ確認**

BigQuery に送信したデータが格納されているか確認します。

```sh
./tutorial/query.sh "<walkthrough-project-id />"
```

Aggregator を手動で実行します。

```sh
gcloud beta run jobs execute \
  aggregator \
  --project "<walkthrough-project-id />" \
  --region us-central1
```

表示された URL にアクセスするとジョブの実行状況が確認できます。
また、次のコマンドでも確認できます。

### **Aggregator 実行**

```sh
gcloud beta run jobs executions list \
  --project "<walkthrough-project-id />" \
  --region us-central1
```

ジョブの終了を待ってから、Cloud Storage にアップロードされたファイルを確認します。

```sh
gsutil ls gs://<walkthrough-project-id />-temperature/
```

最初の実行では直前の1時間 (例えば9:10に実行すると、8:00〜9:00)
が集計されるため、この手順を短時間で終えると通常は空のファイルがアップロードされます。
1時間待つと送信したデータが集計されます。確認する場合は1時間後に次のコマンドを実行して確認してください。


```sh
gsutil cat "$(gsutil ls gs://<walkthrough-project-id />-temperature/ | tail -1)"
```

## おつかれさまでした

以上でデモの構築は終了です。

<walkthrough-conclusion-trophy />

不要になったリソースやプロジェクトは削除してください。
